package postgres

// Shared SQL fragment for case-insensitive alphabetical ordering by name.
const orderByNameAsc = " ORDER BY LOWER(name) ASC, name ASC, id ASC"
