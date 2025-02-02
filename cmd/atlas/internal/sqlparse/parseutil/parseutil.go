// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

// Package parseutil exposes shared functions used by the different parsers.
package parseutil

import (
	"bytes"
	"fmt"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"golang.org/x/exp/slices"
)

// Rename describes rename of a resource.
type Rename struct {
	From, To string
}

// RenameColumn patches DROP/ADD column commands to RENAME.
func RenameColumn(modify *schema.ModifyTable, r *Rename) {
	changes := schema.Changes(modify.Changes)
	i := changes.IndexDropColumn(r.From)
	j := changes.IndexAddColumn(r.To)
	if i != -1 && j != -1 {
		changes[max(i, j)] = &schema.RenameColumn{
			From: changes[i].(*schema.DropColumn).C,
			To:   changes[j].(*schema.AddColumn).C,
		}
		changes.RemoveIndex(min(i, j))
		modify.Changes = changes
	}
}

// RenameIndex patches DROP/ADD index commands to RENAME.
func RenameIndex(modify *schema.ModifyTable, r *Rename) {
	changes := schema.Changes(modify.Changes)
	i := changes.IndexDropIndex(r.From)
	j := changes.IndexAddIndex(r.To)
	if i != -1 && j != -1 {
		changes[max(i, j)] = &schema.RenameIndex{
			From: changes[i].(*schema.DropIndex).I,
			To:   changes[j].(*schema.AddIndex).I,
		}
		changes.RemoveIndex(min(i, j))
		modify.Changes = changes
	}
}

// RenameTable patches DROP/ADD table commands to RENAME.
func RenameTable(changes schema.Changes, r *Rename) schema.Changes {
	i := changes.IndexDropTable(r.From)
	j := changes.IndexAddTable(r.To)
	if i != -1 && j != -1 {
		changes[max(i, j)] = &schema.RenameTable{
			From: changes[i].(*schema.DropTable).T,
			To:   changes[j].(*schema.AddTable).T,
		}
		changes.RemoveIndex(min(i, j))
	}
	return changes
}

// MatchStmtBefore reports if the file contains any statement that matches the predicate before the given position.
func MatchStmtBefore(f migrate.File, pos int, p func(*migrate.Stmt) (bool, error)) (bool, error) {
	stmts, err := StmtDecls(f)
	if err != nil {
		return false, err
	}
	i := slices.IndexFunc(stmts, func(s *migrate.Stmt) bool {
		return s.Pos >= pos
	})
	if i != -1 {
		stmts = stmts[:i]
	}
	for _, s := range stmts {
		m, err := p(s)
		if err != nil {
			return false, err
		}
		if m {
			return true, nil
		}
	}
	return false, nil
}

// StmtDecls returns the statement declarations of a file.
func StmtDecls(f migrate.File) ([]*migrate.Stmt, error) {
	if s, ok := f.(interface {
		StmtDecls() ([]*migrate.Stmt, error)
	}); ok {
		return s.StmtDecls()
	}
	s1, err := f.Stmts()
	if err != nil {
		return nil, err
	}
	s2 := make([]*migrate.Stmt, len(s1))
	for i := range s1 {
		p, err := pos(f, s1[i])
		if err != nil {
			return nil, err
		}
		s2[i] = &migrate.Stmt{Pos: p, Text: s1[i]}
	}
	return s2, nil
}

// pos returns the position of a statement in migration file.
func pos(f migrate.File, stmt string) (int, error) {
	i := bytes.Index(f.Bytes(), []byte(stmt))
	if i == -1 {
		return 0, fmt.Errorf("statement %q was not found in %q", stmt, f.Bytes())
	}
	return i, nil
}

func max(i, j int) int {
	if i > j {
		return i
	}
	return j
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}
