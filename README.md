![](https://avatars.githubusercontent.com/u/80110794?s=200&v=4)

# Morph

Morph is a database migration tool that helps you to apply your migrations. It is written with Go so you can use it from your Go application as well.

## Usage

It can be used as a library or a CLI tool.

### Library

```Go
import (
    "github.com/go-morph/morph"
    bindata "github.com/go-morph/morph/sources/go_bindata"
)

src, err := bindata.WithInstance(&bindata.AssetSource{
    Names: []string{}, // add migration file names
    AssetFunc: func(name string) ([]byte, error) {
        return []byte{}, nil // should return the file contents
    },
})
if err != nil {
    return err
}
defer src.Close()

engine, err := morph.NewFromConnURL(dsn, src, "mysql")
if err != nil {
    return err
}

engine.ApplyAll()

```

### CLI

To install `morph` you can use:

```bash
go install github.com/go-morph/morph/cmd/morph@latest
```

Then you can apply your migrations like below:

```bash
morph apply up --driver postgres --dsn "postgres://user:pass@localhost:5432/mydb?sslmode=disable" --path ./db/migrations/postgres --number 1
```

## Migration Files

The migrations files should have an `up` and `down` versions. The program requires each migration to be reversable. And the naming of the migration should be in the form of following:
```
0000000001_create_user.up.sql
0000000001_create_user.down.sql
```

The first part will determined as the db version and the part between version and `up|down.sql` will be the migration name.

The program requires this naming convention to be followed as it saves the version and names of the migrations. Also, it can rollback migrations with the `down` files.

## LICENSE

[MIT](LICENSE)
