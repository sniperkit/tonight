package mysql

import (
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"github.com/bobinette/tonight/tonight"
)

func Format(user, password, host, port, database string) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		user,
		password,
		host,
		port,
		database,
	)
}

func join(s, sep string, n int) string {
	a := make([]string, n)
	for i := 0; i < n; i++ {
		a[i] = s
	}
	return strings.Join(a, sep)
}

type keepOrder struct {
	idOrder map[uint]int
	tasks   []tonight.Task
}

func (k *keepOrder) Len() int      { return len(k.tasks) }
func (k *keepOrder) Swap(i, j int) { k.tasks[i], k.tasks[j] = k.tasks[j], k.tasks[i] }
func (k *keepOrder) Less(i, j int) bool {
	return k.idOrder[k.tasks[i].ID] < k.idOrder[k.tasks[j].ID]
}
