# scopes

The `scopes` package encapsulates the part of the [auth system](https://impractical.co/auth) that defines access gates and controls which users can open them.

It stores a unique identifier for the access type, a list of default scopes, and whether a user, a client, or both most be whitelisted to use the scope (and if so, it is in charge of maintaining those whitelists).

## Implementation

Scopes consist of an ID, a user policy, a client policy, and whether the scope is a default scope. The ID must uniquely identify the type of access, and should be formatted as a URI. The user policy and client policy are specific strings; for the moment, only `DENY_ALL`, `DEFAULT_DENY`, `DEFAULT_ALLOW`, and `ALLOW_ALL` are used, but that may be expanded in the future.

`DENY_ALL` will deny attempts to use that scope by an client/user. This is useful for deprecated scopes. `DEFAULT_DENY` will deny any request to use the scope by any client/user not in the scope's list, but will allow those in the list to use the scope. `DEFAULT_ALLOW` will deny any request to use the scope by any client/user in the scope's list, but will allow those not in the list to use the scope. `ALLOW_ALL` will allow every client/user to request the scope.

If the scope is marked as a default scope, it will be returned in the list of scopes provided when no scopes are requested.

## Scope

`scopes` is solely responsible for managing the list of scopes and the ACL it needs to determine who and what have the appropriate rights to request a certain scope.

The questions `scopes` is meant to answer for the system include:

  * Are these scopes valid?
  * Which of these scopes can this user grant?
  * Which of these scopes can be granted to this client?
  * What scopes should we grant if none are requested?

The things `scopes` is explicitly not expected to do include:

  * Actually controlling access to anything.
  * Authenticating users.
  * Authenticating clients.
  * Remembering which users have granted which scopes to which clients.
