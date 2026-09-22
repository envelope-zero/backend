# Model Integrity Checks

Semantic integrity of resources is checked in the [models package](../pkg/models/) with [gorm callbacks](https://gorm.io/docs/write_plugins.html#Callbacks).

There are two main reasons for this:

1. Foreign key errors in SQLite do not return any information about the foreign key reference that is failing. For details on why that is, check out [this mail thread](https://sqlite-users.sqlite.narkive.com/dbLQTqwB/sqlite-how-hard-is-it-to-add-the-constraint-name-to-the-foreign-key-constraint-failed-message)
2. We can improve and internationalize error messages passed to users by checking specific error conditions directly and returning the appropriate error messages.
