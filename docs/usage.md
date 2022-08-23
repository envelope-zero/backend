# Usage

## Resources

### Budget

A budget might represent your own personal budget, the budget of your flat share or any other budget that you want to track.

### Account

An account represents an account as an accountant would use it. For example, cash in your wallet, an actual bank account, a credit card are all distinct accounts.
Every other person or company you pay money to also is tracked as an account. This allows for detailed accounting of how much spending occurs towards e.g. a specific supermarket.

### Category

A category is a grouping of envelopes for similar purposes. You might for example want to group envelopes for different saving goals like a new couch, the next vacation etc. in a category called “Saving“.

### Envelope

The key of the budget. Each envelope tracks budget for one specific purpose, e.g. groceries. Any transaction you make for groceries gets linked to this envelope, allowing you to track how much you spent on groceries in a month.

If you do not spend all of the allocated budget, it rolls over to the next month. This allows to allocate a budget for expenses that do not occur that often. Say you have an insurance that is billed yearly for 120€. In this case, you can budget 10€ per month for this insurance. Once the bill arrives, all the needed budget is already allocated.

### Transaction

A transaction represents an actual transaction. Paying for groceries, getting paid your salary, gifting somebody money for their wedding, all of those are transactions. Every transaction must be tied to a source account and destination account.

All transactions that are not income should be tied to an envelope so that you can see what you spend money for.

### Allocation

An allocation is the database representation of the sum that is budgeted for a month for a specific envelope.

## Deleting resources

Envelope Zero soft deletes almost all resources, so you will be able to restore them.

This does not apply to budgeted amounts for a month (which are stored in an _allocation_), those are always permanently deleted.
