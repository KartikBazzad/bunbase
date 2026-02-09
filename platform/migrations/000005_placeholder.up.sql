-- No-op migration: keeps migration version at 5 so the migrator does not fail when DB is already at version 5.
SELECT 1;
