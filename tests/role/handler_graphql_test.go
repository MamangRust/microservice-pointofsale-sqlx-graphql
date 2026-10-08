package role_test

import (
	"testing"

	tests "github.com/MamangRust/microservice-point-of-sale-test"
	"github.com/stretchr/testify/suite"
)

type RoleGraphqlTestSuite struct {
	tests.BaseTestSuite
	gql    *tests.GraphQLClient
	roleID int
}

func (s *RoleGraphqlTestSuite) SetupSuite() {
	s.BaseTestSuite.SetupSuite()

	s.SetupRoleService()

	// Role administration is guarded by @hasRole(roles: ["Admin", "ROLE_ADMIN"]).
	// The harness mounts a permissive checker, so the directive passes through
	// without needing an authenticated caller.
	s.gql = tests.NewGraphQLClient(s.T(), s.Conns, s.RedisClient(), s.Log)
}

func (s *RoleGraphqlTestSuite) TestRoleGraphqlLifecycle() {
	// 1. Create
	create := s.gql.Do(`
		mutation Create($input: CreateRoleInput!) {
			createRole(input: $input) { status message data { id name } }
		}`, map[string]interface{}{"input": map[string]interface{}{"name": "Graphql Role"}})
	created := tests.Payload(s.T(), create, "createRole")
	s.roleID = tests.Int(s.T(), created, "id")
	s.Require().NotZero(s.roleID)
	s.Equal("Graphql Role", tests.Str(s.T(), created, "name"))

	// 2. FindById
	found := s.gql.Do(`
		query Find($input: FindByIdRoleInput!) { findByIdRole(input: $input) { status data { id name } } }`,
		map[string]interface{}{"input": map[string]interface{}{"role_id": s.roleID}})
	s.Equal(s.roleID, tests.Int(s.T(), tests.Payload(s.T(), found, "findByIdRole"), "id"))

	// 3. FindAll
	all := s.gql.Do(`query { findAllRole(input: {page: 1, page_size: 10}) { status data { id } } }`, nil)
	s.NotEmpty(tests.Items(s.T(), all, "findAllRole"))

	// 4. FindByActive / FindByTrashed
	active := s.gql.Do(`query { findByActiveRole(input: {page: 1, page_size: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), active, "findByActiveRole"))

	trashedList := s.gql.Do(`query { findByTrashedRole(input: {page: 1, page_size: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), trashedList, "findByTrashedRole"))

	// 5. FindByUserId
	byUser := s.gql.Do(`
		query ByUser($input: FindByIdUserRoleInput!) { findByUserIdRole(input: $input) { status data { id } } }`,
		map[string]interface{}{"input": map[string]interface{}{"user_id": 1}})
	s.Equal("success", tests.Status(s.T(), byUser, "findByUserIdRole"))

	// 6. Update
	update := s.gql.Do(`
		mutation Update($input: UpdateRoleInput!) {
			updateRole(input: $input) { status data { id name } }
		}`, map[string]interface{}{"input": map[string]interface{}{
		"id":   s.roleID,
		"name": "Graphql Role Updated",
	}})
	s.Equal("Graphql Role Updated", tests.Str(s.T(), tests.Payload(s.T(), update, "updateRole"), "name"))

	// 7. Trash / restore
	trashed := s.gql.Do(`
		mutation Trash($input: FindByIdRoleInput!) { trashedRole(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"role_id": s.roleID}})
	s.Equal("success", tests.Status(s.T(), trashed, "trashedRole"))

	restored := s.gql.Do(`
		mutation Restore($input: FindByIdRoleInput!) { restoreRole(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"role_id": s.roleID}})
	s.Equal("success", tests.Status(s.T(), restored, "restoreRole"))

	// 8. DeletePermanent
	s.gql.Do(`
		mutation Trash($input: FindByIdRoleInput!) { trashedRole(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"role_id": s.roleID}})
	permanent := s.gql.Do(`
		mutation DeletePermanent($input: FindByIdRoleInput!) { deleteRolePermanent(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"role_id": s.roleID}})
	s.Equal("success", tests.Status(s.T(), permanent, "deleteRolePermanent"))

	// 9. Bulk
	restoreAll := s.gql.Do(`mutation { restoreAllRole { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), restoreAll, "restoreAllRole"))
	deleteAll := s.gql.Do(`mutation { deleteAllRolePermanent { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), deleteAll, "deleteAllRolePermanent"))
}

func TestRoleGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(RoleGraphqlTestSuite))
}
