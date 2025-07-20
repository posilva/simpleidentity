# database schema

| Access Pattern                 |-----------PK---------------|-----------SK------------------------|----------GSI1PK--------------|----------GSI1SK--------------|

|..................................................................................................................................|..............................|
| Get Account by Provider ID     | ACTN#<account_id>           | PVDR#<provider>#<provider_id>        PVDR#<provider>#<provider_id> ACNT#<account_id>
| Get Account                    | ACTN#<account_ID>           | ACNT#DATA


