package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/knoebber/dotfile/usererr"
	"github.com/pkg/errors"
)

const (
	fileCountQuery    = "SELECT COUNT(*) FROM files WHERE user_id = ?"
	fileValidateQuery = "SELECT COUNT(*) FROM files WHERE user_id = ? AND alias = ?"
)

// File models the files table.
// It stores the contents of a file at the current revision hash.
//
// Both aliases and paths must be unique for each user.
type File struct {
	ID              int64
	UserID          int64  `validate:"required"`
	Alias           string `validate:"required"` // Friendly name for a file: bashrc
	Path            string `validate:"required"` // Where the file lives: ~/.bashrc
	CurrentRevision string
	Content         []byte `validate:"required"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// FileSummary summarizes a file.
type FileSummary struct {
	Alias      string
	Path       string
	NumCommits int
	UpdatedAt  string
}

// Unique indexes prevent a user from having duplicate alias / path.
func (*File) createStmt() string {
	return `
CREATE TABLE IF NOT EXISTS files(
id                 INTEGER PRIMARY KEY,
user_id            INTEGER NOT NULL REFERENCES users,
alias              TEXT NOT NULL COLLATE NOCASE,
path               TEXT NOT NULL COLLATE NOCASE,
current_revision   TEXT NOT NULL,
content            BLOB NOT NULL,
created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS files_user_index ON files(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS files_user_alias_index ON files(user_id, alias);
CREATE UNIQUE INDEX IF NOT EXISTS files_user_path_index ON files(user_id, path);
`
}

func (f *File) check() error {
	// TODO path should either start with '/' or '~'.
	// Path should be allowed to be empty.
	// When path is empty the templates should display it as 'web-only', and
	// be disallowed from being pulled.
	var count int

	exists, err := fileExists(f.UserID, f.Alias)
	if err != nil {
		return err
	}
	if exists {
		return usererr.Duplicate("File alias", f.Alias)
	}

	if err := checkSize(f.Content, "File "+f.Alias); err != nil {
		return err
	}

	if err := connection.QueryRow(fileCountQuery, f.UserID).Scan(&count); err != nil {
		return errors.Wrapf(err, "counting user %d file", f.UserID)
	}

	if count > maxFilesPerUser {
		return usererr.Invalid("User has maximum amount of files")
	}

	return nil
}

func (f *File) insertStmt(e executor) (sql.Result, error) {
	return e.Exec(`
INSERT INTO files(user_id, alias, path, current_revision, content) VALUES(?, ?, ?, ?, ?)`,
		f.UserID,
		f.Alias,
		f.Path,
		f.CurrentRevision,
		f.Content,
	)
}

func (f *File) scan(row *sql.Row) error {
	return row.Scan(
		&f.ID,
		&f.UserID,
		&f.Alias,
		&f.Path,
		&f.CurrentRevision,
		&f.Content,
		&f.CreatedAt,
		&f.UpdatedAt,
	)
}

// Update updates the alias or path if they are different.
// TODO not using yet - unsure if allowing users to update alias / path is wise.
func (f *File) Update(newAlias, newPath string) error {
	if newAlias == "" || newPath == "" {
		return fmt.Errorf("cannot update to empty value, alias: %#v, path: %#v", newAlias, newPath)
	}

	if f.Alias == newAlias && f.Path == newPath {
		return nil
	}
	_, err := connection.Exec(`
UPDATE files
SET alias = ?, path = ?, updated_at = ?
WHERE file_id = ?
`, newAlias, newPath, time.Now(), f.ID)
	if err != nil {
		return errors.Wrapf(err, "updating file %d to %#v %#v", f.ID, newAlias, newPath)
	}

	return nil
}

func updateContent(tx *sql.Tx, fileID int64, content []byte, hash string) error {
	if err := checkSize(content, "File revision "+hash); err != nil {
		return err
	}

	if _, err := tx.Exec(`
UPDATE files
SET content = ?, current_revision = ?, updated_at = ?
WHERE id = ?
`, content, hash, time.Now(), fileID); err != nil {
		return rollback(tx, errors.Wrapf(err, "updating content in file %d", fileID))
	}
	return nil
}

// GetFileByUsername retrieves a user's file by their username.
func GetFileByUsername(username string, alias string) (*File, error) {
	file := new(File)

	row := connection.QueryRow(`
SELECT files.* FROM files
JOIN users ON user_id = users.id
WHERE username = ? AND alias = ?
`, username, alias)

	if err := file.scan(row); err != nil {
		return nil, errors.Wrapf(err, "querying for user %#v file %#v", username, alias)
	}

	return file, nil
}

// GetFilesByUsername gets a summary of all a users files.
func GetFilesByUsername(username string) ([]FileSummary, error) {
	result := []FileSummary{}
	rows, err := connection.Query(`
SELECT alias,
       path,
       COUNT(commits.id) AS num_commits,
       updated_at
FROM files
JOIN users on user_id = users.id
JOIN commits on file_id = files.id
WHERE username = ?
GROUP BY files.id`, username)
	if err != nil {
		return nil, errors.Wrapf(err, "querying user %#v files", username)
	}
	defer rows.Close()

	for rows.Next() {
		f := FileSummary{}
		updatedAt := time.Time{}

		if err := rows.Scan(
			&f.Alias,
			&f.Path,
			&f.NumCommits,
			&updatedAt,
		); err != nil {
			return nil, errors.Wrapf(err, "scanning files for user %#v", username)
		}

		f.UpdatedAt = formatTime(updatedAt)
		result = append(result, f)
	}

	return result, nil
}

func getFileByUserID(userID int64, alias string) (*File, error) {
	file := new(File)

	row := connection.
		QueryRow("SELECT * FROM files WHERE user_id = ? AND alias = ?", userID, alias)

	if err := file.scan(row); err != nil {
		return nil, errors.Wrapf(err, "querying for user %d file %#v", userID, alias)
	}

	return file, nil
}

func fileExists(userID int64, alias string) (bool, error) {
	var count int

	err := connection.QueryRow(fileValidateQuery, userID, alias).Scan(&count)
	if err != nil {
		return false, errors.Wrapf(err, "checking if file %#v exists for user %d", alias, userID)
	}
	return count > 0, nil
}
