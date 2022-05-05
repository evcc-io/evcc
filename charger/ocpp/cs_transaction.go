package ocpp

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

func (cs *CS) loadLastTransaction(id string) (Transaction, error) {
	var txn Transaction
	fileName := id + "_transaction.yaml"
	_, err := os.Stat(fileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return txn, nil
		}

		return txn, err
	}
	txnFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		return txn, fmt.Errorf("failed to load txnFile for %s: %w", id, err)
	}

	err = yaml.Unmarshal(txnFile, &txn)
	if err != nil {
		return txn, fmt.Errorf("failed to unmarshal transactions for %s: %s", id, err)
	}

	return txn, nil
}

func (cs *CS) loadTransactionFile(id string) (Transactions, error) {
	var txn Transactions

	fileName := id + "_transactions.yaml"

	_, err := os.Stat(fileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return txn, nil
		}

		return nil, err
	}

	txnFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to load txnFile for %s: %w", id, err)
	}

	err = yaml.Unmarshal(txnFile, &txn)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal transactions for %s: %s", id, err)
	}

	return txn, nil
}
