package database

import "strings"

// splitSQLStatements 将 PostgreSQL 脚本按「括号深度为 0 时的分号」切分为多条语句，
// 忽略单引号字符串内的括号与分号（含 SQL 标准 '' 转义单引号）。用于执行嵌入的多语句迁移。
func splitSQLStatements(s string) []string {
	var stmts []string
	var b strings.Builder
	depth := 0
	inSingle := false
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if inSingle {
			if c == '\'' {
				if i+1 < len(runes) && runes[i+1] == '\'' {
					b.WriteString("''")
					i++
					continue
				}
				inSingle = false
			}
			b.WriteRune(c)
			continue
		}
		if c == '\'' {
			inSingle = true
			b.WriteRune(c)
			continue
		}
		if c == '(' {
			depth++
			b.WriteRune(c)
			continue
		}
		if c == ')' {
			if depth > 0 {
				depth--
			}
			b.WriteRune(c)
			continue
		}
		if c == ';' && depth == 0 {
			stmt := strings.TrimSpace(b.String())
			if stmt != "" && !isOnlySQLLineComments(stmt) {
				stmts = append(stmts, stmt)
			}
			b.Reset()
			continue
		}
		b.WriteRune(c)
	}
	last := strings.TrimSpace(b.String())
	if last != "" && !isOnlySQLLineComments(last) {
		stmts = append(stmts, last)
	}
	return stmts
}

func isOnlySQLLineComments(s string) bool {
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if !strings.HasPrefix(t, "--") {
			return false
		}
	}
	return true
}
