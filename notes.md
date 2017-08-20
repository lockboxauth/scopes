* Bootstrap admin requires a special signed request.
  * Keep a special RSA key pair that's used only to authenticate for ACL.
  * Private key is never deployed anywhere.
  * Public key is given to the scopes service.
  * No ACL modification can happen without signing them using this key pair.
  * Signing must sign the actual JSON of the request, to ensure no modification happens in-flight.
* ACL modifications are as simple as possible:
  * Can set the default set of scopes that is granted when no scopes are requested.
  * Can whitelist users to use specific scopes.
  * Can whitelist clients to use specific scopes.
  * Can set the valid set of scopes.
