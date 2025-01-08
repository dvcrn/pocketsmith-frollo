# Frollo importer for pocketsmith

A simple tool to import transactions from Frollo into Pocketsmith.

# Using environment variables
export FROLLO_USERNAME=your_username
export FROLLO_PASSWORD=your_password
export POCKETSMITH_TOKEN=your_token
export ACCOUNTS_TO_SYNC=account_id1,account_id2

./pocketsmith-frollo

# Or using command line flags
./pocketsmith-frollo -username=your_username -password=your_password -token=your_token -accounts=account_id1,account_id2


The tool will:
- Login to Frollo and fetch transactions
- Create institutions/accounts in Pocketsmith if they don't exist
- Import transactions that don't already exist
- Update account balances
