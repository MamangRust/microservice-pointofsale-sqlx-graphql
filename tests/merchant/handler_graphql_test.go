package merchant_test

import (
	"testing"

	tests "github.com/MamangRust/microservice-point-of-sale-test"
	"github.com/stretchr/testify/suite"
)

type MerchantGraphqlTestSuite struct {
	tests.BaseTestSuite
	gql        *tests.GraphQLClient
	userID     int
	merchantID int
}

func (s *MerchantGraphqlTestSuite) SetupSuite() {
	s.BaseTestSuite.SetupSuite()

	s.SetupUserService()
	s.SetupMerchantService()

	s.userID = s.SeedUser(s.Ctx)

	s.gql = tests.NewGraphQLClient(s.T(), s.Conns, s.RedisClient(), s.Log)
}

func (s *MerchantGraphqlTestSuite) TestMerchantGraphqlLifecycle() {
	// 1. Create
	create := s.gql.Do(`
		mutation Create($input: CreateMerchantInput!) {
			createMerchant(input: $input) { status message data { id name status } }
		}`, map[string]interface{}{"input": map[string]interface{}{
		"user_id":       s.userID,
		"name":          "Graphql Merchant",
		"description":   "Graphql Description",
		"address":       "Graphql Address",
		"contact_email": "merchant@example.com",
		"contact_phone": "123456789",
		"status":        "active",
	}})
	created := tests.Payload(s.T(), create, "createMerchant")
	s.merchantID = tests.Int(s.T(), created, "id")
	s.Require().NotZero(s.merchantID)
	s.Equal("Graphql Merchant", tests.Str(s.T(), created, "name"))

	// 2. FindById
	found := s.gql.Do(`
		query Find($input: FindByIdMerchantInput!) { findByIdMerchant(input: $input) { status data { id } } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.merchantID}})
	s.Equal(s.merchantID, tests.Int(s.T(), tests.Payload(s.T(), found, "findByIdMerchant"), "id"))

	// 3. FindAll
	all := s.gql.Do(`query { findAllMerchant(input: {page: 1, page_size: 10}) { status data { id } } }`, nil)
	s.NotEmpty(tests.Items(s.T(), all, "findAllMerchant"))

	// 4. FindByActive
	active := s.gql.Do(`query { findByActiveMerchant(input: {page: 1, page_size: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), active, "findByActiveMerchant"))

	// 5. Update
	update := s.gql.Do(`
		mutation Update($input: UpdateMerchantInput!) {
			updateMerchant(input: $input) { status data { id name } }
		}`, map[string]interface{}{"input": map[string]interface{}{
		"merchant_id":   s.merchantID,
		"user_id":       s.userID,
		"name":          "Graphql Merchant Updated",
		"description":   "Updated Description",
		"address":       "Updated Address",
		"contact_email": "updated@example.com",
		"contact_phone": "987654321",
		"status":        "active",
	}})
	s.Equal("Graphql Merchant Updated", tests.Str(s.T(), tests.Payload(s.T(), update, "updateMerchant"), "name"))

	// 6. Trash / findByTrashed / restore
	trashed := s.gql.Do(`
		mutation Trash($input: FindByIdMerchantInput!) { trashedMerchant(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.merchantID}})
	s.Equal("success", tests.Status(s.T(), trashed, "trashedMerchant"))

	trashedList := s.gql.Do(`query { findByTrashedMerchant(input: {page: 1, page_size: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), trashedList, "findByTrashedMerchant"))

	restored := s.gql.Do(`
		mutation Restore($input: FindByIdMerchantInput!) { restoreMerchant(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.merchantID}})
	s.Equal("success", tests.Status(s.T(), restored, "restoreMerchant"))

	// 7. DeletePermanent
	s.gql.Do(`
		mutation Trash($input: FindByIdMerchantInput!) { trashedMerchant(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.merchantID}})
	permanent := s.gql.Do(`
		mutation DeletePermanent($input: FindByIdMerchantInput!) { deleteMerchantPermanent(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.merchantID}})
	s.Equal("success", tests.Status(s.T(), permanent, "deleteMerchantPermanent"))

	// 8. Bulk
	restoreAll := s.gql.Do(`mutation { restoreAllMerchant { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), restoreAll, "restoreAllMerchant"))
	deleteAll := s.gql.Do(`mutation { deleteAllMerchantPermanent { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), deleteAll, "deleteAllMerchantPermanent"))
}

func TestMerchantGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(MerchantGraphqlTestSuite))
}
