#!/usr/bin/env zsh

http post localhost:8080/v1/budgets name=Morre
http post localhost:8080/v1/budgets/1/accounts name='Bank Account'
http post localhost:8080/v1/budgets/1/accounts name=Cash
http post localhost:8080/v1/budgets/1/accounts name=Netflix external:=true

http post localhost:8080/v1/budgets/1/categories name='Laufende Kosten'
http post localhost:8080/v1/budgets/1/categories/1/envelopes name='Abos'

http post localhost:8080/v1/budgets/1/categories/1/envelopes/1/allocations month:=1 year:=2022 amount:=20.99
http post localhost:8080/v1/budgets/1/categories/1/envelopes/1/allocations month:=2 year:=2022 amount:=30.00
http post localhost:8080/v1/budgets/1/categories/1/envelopes/1/allocations month:=3 year:=2022 amount:=47.12

http post localhost:8080/v1/budgets/1/transactions amount='17.99' sourceAccountId:=1 destinationAccountId:=3 date='2022-02-25T00:00:00Z' envelopeId:=1
http post localhost:8080/v1/budgets/1/transactions amount='17.99' sourceAccountId:=1 destinationAccountId:=3 date='2022-01-25T00:00:00Z' envelopeId:=1 completed:=true
http post localhost:8080/v1/budgets/1/transactions amount='17.99' sourceAccountId:=1 destinationAccountId:=3 date='2021-12-27T00:00:00Z' envelopeId:=1 completed:=true
http post localhost:8080/v1/budgets/1/transactions amount='17.99' sourceAccountId:=1 destinationAccountId:=3 date='2021-11-25T00:00:00Z' envelopeId:=1 completed:=true
http post localhost:8080/v1/budgets/1/transactions amount='17.99' sourceAccountId:=1 destinationAccountId:=3 date='2021-10-26T00:00:00Z' envelopeId:=1 completed:=true

