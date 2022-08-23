# Database schema

This document aims at developers who need to understand how the database schema looks. If you are looking for usage hints, go to [the usage guide](usage.md).

## Entity Relationship diagram

<!-- Use https://mermaid.live for easier editing of this diagram -->

```mermaid
erDiagram
    Budget ||--o{ Account : has
    Budget ||--o{ Category : has
    Budget ||--o{ Transaction : "has, see #308"
    Transaction }o..o| Envelope : "has"
    Transaction }o--|| Account : "has source"
    Transaction }o--|| Account : "has destination"
    Category ||--o{ Envelope : has
    Envelope ||--o{ Allocation : "has max. 1 per Month"
```
