package global

import "context"

func SearchUsers(ctx context.Context, query string) []User {
	res := System.SearchUsers(ctx, query)
	users := make([]User, len(res))
	for i, r := range res {
		users[i] = User{r}
	}
	return users
}
