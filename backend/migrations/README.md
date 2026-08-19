# Database migrations

FlagStack will keep ordered PostgreSQL schema migrations in this directory.

The first product-domain migration will be added with the initial organisation/project/environment/flag model. Migration tooling will be selected alongside that schema so the repository does not commit to a migration dependency before it is actually needed.
