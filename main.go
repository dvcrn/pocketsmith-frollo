package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dvcrn/pocketsmith-frollo/frollo"
	"github.com/dvcrn/pocketsmith-go"
)

const accsToSync = "1657651,1657652"

type Config struct {
	FrolloUsername   string
	FrolloPassword   string
	PocketsmithToken string
	AccountsToSync   string
	NumTransactions  int
}

func getConfig() *Config {
	config := &Config{}

	// Define command-line flags
	flag.StringVar(&config.FrolloUsername, "username", os.Getenv("FROLLO_USERNAME"), "Frollo username")
	flag.StringVar(&config.FrolloPassword, "password", os.Getenv("FROLLO_PASSWORD"), "Frollo password")
	flag.StringVar(&config.PocketsmithToken, "token", os.Getenv("POCKETSMITH_TOKEN"), "Pocketsmith API token")
	flag.StringVar(&config.AccountsToSync, "accounts", os.Getenv("ACCOUNTS_TO_SYNC"), "Comma-separated list of account IDs to sync")
	flag.Parse()

	// Validate required fields
	if config.FrolloUsername == "" {
		fmt.Println("Error: Frollo username is required. Set via -username flag or FROLLO_USERNAME environment variable")
		os.Exit(1)
	}
	if config.FrolloPassword == "" {
		fmt.Println("Error: Frollo password is required. Set via -password flag or FROLLO_PASSWORD environment variable")
		os.Exit(1)
	}
	if config.PocketsmithToken == "" {
		fmt.Println("Error: Pocketsmith token is required. Set via -token flag or POCKETSMITH_TOKEN environment variable")
		os.Exit(1)
	}
	if config.AccountsToSync == "" {
		fmt.Println("Error: accounts to sync is required. Set it via -accounts flag or ACCOUNTS_TO_SYNC environment variable")
		os.Exit(1)
	}

	return config
}

func main() {
	config := getConfig()

	ps := pocketsmith.NewClient(config.PocketsmithToken)

	currentUser, err := ps.GetCurrentUser()
	if err != nil {
		panic(err)
	}

	c := frollo.NewClient()
	_, err = c.Login(config.FrolloUsername, config.FrolloPassword)
	if err != nil {
		panic(err)
	}

	// kick off sync
	c.SyncAccounts()

	for _, accId := range strings.Split(config.AccountsToSync, ",") {
		acc, err := c.GetAccount(accId)
		if err != nil {
			panic(err)
		}

		if acc.AccountStatus != "active" {
			fmt.Printf("only active accounts are synced, skipping '%s' (status %s)\n", acc.AccountName, acc.AccountStatus)
			continue
		}

		if acc.AccountAttributes.AccountType != "bank_account" && acc.AccountAttributes.AccountType != "savings" {
			fmt.Printf("only bank accounts are currently supported, skipping '%s' (type %s)\n", acc.AccountName, acc.AccountAttributes.AccountType)
			continue
		}

		fmt.Printf("syncing account '%s'\n", acc.AccountName)

		// from now, keep going backwards by 6 months until we get no more results
		fromDate := time.Now()
		allTxs := []*frollo.FrolloTransaction{}
		for {
			fmt.Printf("searching %s -> %s\n", fromDate.Format("2006-01-02"), fromDate.AddDate(0, -6, 0).Format("2006-01-02"))

			toDate := fromDate.AddDate(0, -12, 0)
			txs, err := c.GetTransactions(accId, toDate, fromDate)
			if err != nil {
				panic(err)
			}

			if len(txs) == 0 {
				break
			}

			fromDate = toDate
			allTxs = append(allTxs, txs...)
		}

		if len(allTxs) == 0 {
			// no txs so we can skip this
			continue
		}

		// try to find this account
		foundAccount, err := ps.FindAccountByName(currentUser.ID, acc.AccountName)
		if err == pocketsmith.ErrNotFound {
			// Find or create institution
			foundInstitution, err := ps.FindInstitutionByName(currentUser.ID, acc.Provider.Name)
			if err == pocketsmith.ErrNotFound {
				foundInstitution, err = ps.CreateInstitution(currentUser.ID, acc.Provider.Name, strings.ToLower(acc.PrimaryBalance.Currency))
				if err != nil {
					panic(err)
				}

				fmt.Printf("Created institution '%s' with ID %d", foundInstitution.Title, foundInstitution.ID)
			} else if err != nil {
				panic(err)
			}

			// Create account in institution
			foundAccount, err = ps.CreateAccount(currentUser.ID, foundInstitution.ID, acc.AccountName, strings.ToLower(acc.PrimaryBalance.Currency), pocketsmith.AccountTypeBank)
			if err != nil {
				panic(err)
			}
		} else if err != nil {
			panic(err)
		}

		fmt.Printf("Account %s has %d transactions\n", acc.AccountName, len(allTxs))

		// sort allTxs by date
		sort.Slice(allTxs, func(i, j int) bool {
			date1, _ := time.Parse("2006-01-02", allTxs[i].TransactionDate)
			date2, _ := time.Parse("2006-01-02", allTxs[j].TransactionDate)
			return date1.After(date2)
		})

		consecutiveAlreadyProcessed := 0
		for _, tx := range allTxs {
			if consecutiveAlreadyProcessed > 10 {
				fmt.Printf("  already processed 10 consecutive transactions, stopping\n")
				break
			}

			fmt.Printf("  %s: %s\n", tx.TransactionDate, tx.Description.Original)

			// check if we already have this transaction
			txDateParsed, err := time.Parse("2006-01-02", tx.TransactionDate) // is 2025-01-07
			if err != nil {
				panic(err)
			}

			foundTxs, err := ps.SearchTransactionsByMemo(foundAccount.PrimaryTransactionAccount.ID, txDateParsed, tx.Reference)
			if err != nil {
				log.Printf("error searching for transactions: %v", err)
				continue
			}

			if len(foundTxs) > 0 {
				fmt.Printf("  already have this transaction, skipping\n")
				consecutiveAlreadyProcessed++
				continue
			} else {
				consecutiveAlreadyProcessed = 0
			}

			amountParsed, err := strconv.ParseFloat(tx.Amount.Amount, 64)
			if err != nil {
				panic(err)
			}

			psTx := &pocketsmith.Transaction{
				Payee:      tx.Description.Original,
				Amount:     amountParsed,
				Date:       tx.TransactionDate,
				IsTransfer: strings.Contains(tx.Type, "transfer"),
				Memo:       tx.Reference,
			}

			if _, err := ps.AddTransaction(foundAccount.PrimaryTransactionAccount.ID, psTx); err != nil {
				log.Printf("error adding transaction: %v", err)
				continue
			}
		}

		// check balance on pocketsmith
		account, err := ps.FindAccountByName(currentUser.ID, foundAccount.PrimaryTransactionAccount.Name)
		if err != nil {
			log.Printf("error finding account: %v", err)
			continue
		}

		convertedBalance, err := strconv.ParseFloat(acc.CurrentBalance.Amount, 64)
		if err != nil {
			log.Printf("error converting balance: %v", err)
			continue
		}

		if account.CurrentBalance < convertedBalance {
			log.Printf("updating balance from %f to %f", account.CurrentBalance, convertedBalance)
			if _, err := ps.UpdateTransactionAccount(foundAccount.PrimaryTransactionAccount.ID, foundAccount.PrimaryTransactionAccount.Institution.ID, convertedBalance, time.Now().Format("2006-01-02")); err != nil {
				log.Printf("error updating balance: %v", err)
			}
		}
	}
}
