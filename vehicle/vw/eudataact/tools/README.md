# EU Data Act live diagnostic

This command performs the real EU Data Act login, lists the account's linked
vehicles, and exercises the newest dataset path for one vehicle.

Run it from the repository root:

```sh
EUDA_EMAIL='you@example.com' EUDA_PASSWORD='secret' \
  go run ./vehicle/vw/eudataact/tools -brand Volkswagen
```

The e-mail can also be passed as the first positional argument. Use `-vin` to
select a vehicle when the account has more than one:

```sh
EUDA_PASSWORD='secret' go run ./vehicle/vw/eudataact/tools \
  -brand Audi -vin WVWZZZ... you@example.com
```

If `EUDA_PASSWORD` is omitted from a terminal, the command prompts without
echoing the password. Credentials are not included in diagnostic output.
