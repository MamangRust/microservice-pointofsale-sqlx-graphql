package category_test

import (
	"testing"

	tests "github.com/MamangRust/microservice-point-of-sale-test"
	"github.com/stretchr/testify/suite"
)

type CategoryGraphqlTestSuite struct {
	tests.BaseTestSuite
	gql        *tests.GraphQLClient
	categoryID int
}

func (s *CategoryGraphqlTestSuite) SetupSuite() {
	s.BaseTestSuite.SetupSuite()

	s.SetupCategoryService()

	s.gql = tests.NewGraphQLClient(s.T(), s.Conns, s.RedisClient(), s.Log)
}

func (s *CategoryGraphqlTestSuite) TestCategoryGraphqlLifecycle() {
	// 1. Create
	create := s.gql.Do(`
		mutation Create($input: CreateCategoryRequest!) {
			createCategory(input: $input) { status message data { id name description } }
		}`, map[string]interface{}{"input": map[string]interface{}{
		"name":        "Graphql Category",
		"description": "Graphql Description",
	}})
	created := tests.Payload(s.T(), create, "createCategory")
	s.categoryID = tests.Int(s.T(), created, "id")
	s.Require().NotZero(s.categoryID)
	s.Equal("Graphql Category", tests.Str(s.T(), created, "name"))

	// 2. FindById
	found := s.gql.Do(`
		query Find($input: FindByIdCategoryRequest!) { findByIdCategory(input: $input) { status data { id } } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.categoryID}})
	s.Equal(s.categoryID, tests.Int(s.T(), tests.Payload(s.T(), found, "findByIdCategory"), "id"))

	// 3. FindAll
	all := s.gql.Do(`query { findAllCategory(input: {page: 1, page_size: 10}) { status data { id } } }`, nil)
	s.NotEmpty(tests.Items(s.T(), all, "findAllCategory"))

	// 4. FindByActive / FindByTrashed
	active := s.gql.Do(`query { findByActiveCategory(input: {page: 1, page_size: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), active, "findByActiveCategory"))

	// 5. Update
	update := s.gql.Do(`
		mutation Update($input: UpdateCategoryRequest!) {
			updateCategory(input: $input) { status data { id name } }
		}`, map[string]interface{}{"input": map[string]interface{}{
		"category_id": s.categoryID,
		"name":        "Graphql Category Updated",
		"description": "Updated Description",
	}})
	s.Equal("Graphql Category Updated", tests.Str(s.T(), tests.Payload(s.T(), update, "updateCategory"), "name"))

	// 6. Trash / findByTrashed / restore
	trashed := s.gql.Do(`
		mutation Trash($input: FindByIdCategoryRequest!) { trashedCategory(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.categoryID}})
	s.Equal("success", tests.Status(s.T(), trashed, "trashedCategory"))

	trashedList := s.gql.Do(`query { findByTrashedCategory(input: {page: 1, page_size: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), trashedList, "findByTrashedCategory"))

	restored := s.gql.Do(`
		mutation Restore($input: FindByIdCategoryRequest!) { restoreCategory(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.categoryID}})
	s.Equal("success", tests.Status(s.T(), restored, "restoreCategory"))

	// 7. DeletePermanent
	s.gql.Do(`
		mutation Trash($input: FindByIdCategoryRequest!) { trashedCategory(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.categoryID}})
	permanent := s.gql.Do(`
		mutation DeletePermanent($input: FindByIdCategoryRequest!) { deleteCategoryPermanent(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.categoryID}})
	s.Equal("success", tests.Status(s.T(), permanent, "deleteCategoryPermanent"))

	// 8. Bulk
	restoreAll := s.gql.Do(`mutation { restoreAllCategory { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), restoreAll, "restoreAllCategory"))
	deleteAll := s.gql.Do(`mutation { deleteAllCategoryPermanent { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), deleteAll, "deleteAllCategoryPermanent"))
}

func TestCategoryGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(CategoryGraphqlTestSuite))
}
