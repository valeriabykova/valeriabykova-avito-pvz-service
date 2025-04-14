#!/bin/bash
goose postgres 'postgres://user:password@localhost:5432/postgres_db?sslmode=disable' -dir db/migrations up