#!/usr/bin/env zsh

http post localhost:8080/v1/budgets name=Morre
http post localhost:8080/v1/budgets/1/accounts name='Bank Account' visible:=true onBudget:=true
http post localhost:8080/v1/budgets/1/accounts name=Cash visible:=true onBudget:=true
http post localhost:8080/v1/budgets/1/accounts name=Netflix

http post localhost:8080/v1/budgets/1/categories name='Laufende Kosten'
http post localhost:8080/v1/budgets/1/categories/1/envelopes name='Abos'

http post localhost:8080/v1/budgets/1/transactions amount='17.99' sourceAccountId:=1 destinationAccountId:=3 date='2022-02-25T00:00:00Z' envelopeId:=1
http post localhost:8080/v1/budgets/1/transactions amount='17.99' sourceAccountId:=1 destinationAccountId:=3 date='2022-01-25T00:00:00Z' envelopeId:=1
http post localhost:8080/v1/budgets/1/transactions amount='17.99' sourceAccountId:=1 destinationAccountId:=3 date='2021-12-27T00:00:00Z' envelopeId:=1
http post localhost:8080/v1/budgets/1/transactions amount='17.99' sourceAccountId:=1 destinationAccountId:=3 date='2021-11-25T00:00:00Z' envelopeId:=1
http post localhost:8080/v1/budgets/1/transactions amount='17.99' sourceAccountId:=1 destinationAccountId:=3 date='2021-10-26T00:00:00Z' envelopeId:=1

