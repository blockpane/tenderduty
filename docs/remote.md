# Loading Remote Configurations

It is possible to load a configuration file over HTTP/HTTPS. This is useful for remote/containerized servers where it is possible to safely pass in a decryption secret via environment variables. This requires first encrypting the file with a password. **Tenderduty will refuse to remotely load a plaintext configuration**.

*Note: Tenderduty does not currently poll the remote config file for changes, and will need to be manually restarted to pick up changes if the file is modified.*

The [encryption](../td2/encryption.go) used is [AES-256](https://en.wikipedia.org/wiki/Advanced_Encryption_Standard) with a random IV, and an authenticated [SHA-256 HMAC](https://en.wikipedia.org/wiki/HMAC). The encryption key and HMAC key are generated using [Argon2id](https://en.wikipedia.org/wiki/Argon2) with a random 32 byte salt, a memory cost of 64MB, and time-constant of one. If you use a good password, brute-force decryption of the file should be difficult.

## Generating the Encrypted Config

Once you have a suitably working configuration, you can convert it to an encrypted file by using tenderduty itself. The simplest way is to run `tenderduty -encrypt` in the same directory as the `config.yml` file. This will prompt for a password (must be at 8 characters or more and not be a common password.) Running this command will output a base64 encoded file at `config.yml.asc`, but the output file can be specified using the `-encrypted-config` flag.

Example:

```
$ ./tenderduty -encrypt
Please enter the encryption password:
2022/07/04 15:32:08 wrote 8856 bytes to encrypted file config.yml.asc
```

## Restoring the Encrypted Config

If you have an encrypted config file, and want to decrypt it, you can inversely use the `-decrypt` flag. Warning: by default this will write to `config.yml` so if you want to retain an existing config use the `-f` flag to specify where the decrypted config should be written.

Example:

```
$ ./tenderduty -decrypt -f decrypted.yml -encrypted-config config.yml.asc
Please enter the encryption password:
2022/07/04 15:36:57 wrote 6549 bytes to decrypted file decrypted.yml
```

## Using a Remotely Hosted Config File

At startup, tenderduty will check the prefix of the config file setting, and if it begins with `http://` or `https://` it will attempt to retrieve the file from a web server. The config file location can be specified on the command line using the `-f` flag, or by setting the `CONFIG` environment variable.

Tendermint will not attempt to use a remote configuration if a password is not set. This can be done by either supplying the `-password` flag or via the `PASSWORD` environment variable. Using the environment variable is slightly safer because **it is cleared once it has been read**, and other users on a system will be able to see the password by looking at running processes if the `-password` flag is used.
