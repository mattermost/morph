# Foundation

A framework to write simple database migration tests.

## Install

```
go get github.com/mgdelacroix/foundation
```

## Usage

To start using foundation, you need to implement the `Migrator`
interface, describing how your tool manages migrations and what are
the intermediate steps (generally data migrations), if any, that need
to run at the end of each migration step:

```go
type Migrator interface {
	DB() *sql.DB
	DriverName() string
	Setup() error
	MigrateToStep(step int) error
	Interceptors() map[int]func() error
	TearDown() error
}

interceptors := map[int]func() error{
    // function that will run after step 6
    6: func() err {
        return myStore.RunDataMigration()
    },
}
```

With the interface implemented, you can use `foundation` in your tests
to load fixtures, set the database on a specific state and then run
your assertions:

```go
t.Run("migration should link book 1 with its author", func(t *testing.T) {
	f := foundation.New(t, migrator).
		// runs migrations up to and including 5
		MigrateToStep(5).
		// loads the SQL of the file
		ExecFile("./myfixtures.sql").
		// runs migration 6 and its interceptor function
		MigrateToStep(6)
	defer f.TearDown()

	book := struct{ID int; AuthorID int}{}

	err := f.DB().Get(&book, "SELECT id, authorID FROM books")
	require.NoError(t, err)
	require.Equal(t, 1, book.ID)
	require.Equal(t, 3, book.AuthorID)
})

t.Run("test specifically the interceptor 6", func(t *testing.T) {
	f := foundation.New(t, migrator).
		MigrateToStepSkippingLastInterceptor(6).
		ExecFile("./myfixtures.sql").
		RunInterceptor(6)
	defer f.TearDown()

	// ...
})
```
