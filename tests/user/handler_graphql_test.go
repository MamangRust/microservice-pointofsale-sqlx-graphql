package user_test

import (
	"testing"

	tests "github.com/MamangRust/microservice-point-of-sale-test"
	"github.com/stretchr/testify/suite"
)

type UserGraphqlTestSuite struct {
	tests.BaseTestSuite
	gql    *tests.GraphQLClient
	userID int
	email  string
}

func (s *UserGraphqlTestSuite) SetupSuite() {
	s.BaseTestSuite.SetupSuite()

	s.SetupRoleService()
	s.SetupUserService()

	// The user service grants this role on create, so it must exist first.
	_, err := s.SQLxDB().ExecContext(s.Ctx,
		`INSERT INTO roles (role_name, created_at, updated_at)
		 VALUES ('Admin Access 1', current_timestamp, current_timestamp)
		 ON CONFLICT (role_name) DO NOTHING`)
	s.Require().NoError(err)

	s.gql = tests.NewGraphQLClient(s.T(), s.Conns, s.RedisClient(), s.Log)
}

func (s *UserGraphqlTestSuite) TestUserGraphqlLifecycle() {
	s.email = "graphql.user@example.com"

	// 1. Create
	create := s.gql.Do(`
		mutation Create($input: CreateUserInput!) {
			createUser(input: $input) { status message data { id firstname lastname email } }
		}`, map[string]interface{}{"input": map[string]interface{}{
		"firstname":        "Graphql",
		"lastname":         "User",
		"email":            s.email,
		"password":         "password123",
		"confirm_password": "password123",
	}})
	created := tests.Payload(s.T(), create, "createUser")
	s.userID = tests.Int(s.T(), created, "id")
	s.Require().NotZero(s.userID)
	s.Equal(s.email, tests.Str(s.T(), created, "email"))

	// 2. FindById
	found := s.gql.Do(`
		query Find($input: FindByIdUserInput!) { findByIdUser(input: $input) { status data { id email } } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.userID}})
	s.Equal(s.userID, tests.Int(s.T(), tests.Payload(s.T(), found, "findByIdUser"), "id"))

	// 3. FindAll
	all := s.gql.Do(`query { findAllUsers(input: {page: 1, page_size: 10}) { status data { id } } }`, nil)
	s.NotEmpty(tests.Items(s.T(), all, "findAllUsers"))

	// 4. FindByActive
	active := s.gql.Do(`query { findByActiveUsers(input: {page: 1, page_size: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), active, "findByActiveUsers"))

	// 5. Update
	update := s.gql.Do(`
		mutation Update($input: UpdateUserInput!) {
			updateUser(input: $input) { status data { id firstname } }
		}`, map[string]interface{}{"input": map[string]interface{}{
		"id":               s.userID,
		"firstname":        "Updated",
		"lastname":         "User",
		"email":            s.email,
		"password":         "password123",
		"confirm_password": "password123",
	}})
	s.Equal("Updated", tests.Str(s.T(), tests.Payload(s.T(), update, "updateUser"), "firstname"))

	// 6. Trash / findByTrashed / restore
	trashed := s.gql.Do(`
		mutation Trash($input: FindByIdUserInput!) { trashedUser(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.userID}})
	s.Equal("success", tests.Status(s.T(), trashed, "trashedUser"))

	trashedList := s.gql.Do(`query { findByTrashedUsers(input: {page: 1, page_size: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), trashedList, "findByTrashedUsers"))

	restored := s.gql.Do(`
		mutation Restore($input: FindByIdUserInput!) { restoreUser(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.userID}})
	s.Equal("success", tests.Status(s.T(), restored, "restoreUser"))

	// 7. DeletePermanent
	s.gql.Do(`
		mutation Trash($input: FindByIdUserInput!) { trashedUser(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.userID}})
	permanent := s.gql.Do(`
		mutation DeletePermanent($input: FindByIdUserInput!) { deleteUserPermanent(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.userID}})
	s.Equal("success", tests.Status(s.T(), permanent, "deleteUserPermanent"))

	// 8. Bulk
	restoreAll := s.gql.Do(`mutation { restoreAllUser { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), restoreAll, "restoreAllUser"))
	deleteAll := s.gql.Do(`mutation { deleteAllUserPermanent { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), deleteAll, "deleteAllUserPermanent"))
}

func TestUserGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(UserGraphqlTestSuite))
}
