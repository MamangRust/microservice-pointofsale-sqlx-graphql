package product_test

import (
	"testing"

	tests "github.com/MamangRust/microservice-point-of-sale-test"
	"github.com/stretchr/testify/suite"
)

type ProductGraphqlTestSuite struct {
	tests.BaseTestSuite
	gql        *tests.GraphQLClient
	userID     int
	categoryID int
	merchantID int
	productID  int
}

func (s *ProductGraphqlTestSuite) SetupSuite() {
	s.BaseTestSuite.SetupSuite()

	// SetupProductService chains the category and merchant services it depends on.
	s.SetupProductService()

	s.userID = s.SeedUser(s.Ctx)
	s.categoryID = s.SeedCategory(s.Ctx)
	s.merchantID = s.SeedMerchant(s.Ctx, s.userID)

	s.gql = tests.NewGraphQLClient(s.T(), s.Conns, s.RedisClient(), s.Log)
}

func (s *ProductGraphqlTestSuite) TestProductGraphqlLifecycle() {
	// 1. Create (multipart: the image field is an Upload scalar)
	create := s.gql.DoUpload(`
		mutation Create($input: CreateProductInput!) {
			createProduct(input: $input) { status message data { id name price } }
		}`, map[string]interface{}{"input": map[string]interface{}{
		"merchantId":   s.merchantID,
		"categoryId":   s.categoryID,
		"name":         "Graphql Product",
		"description":  "Graphql Description",
		"price":        1000,
		"countInStock": 10,
		"brand":        "Graphql Brand",
		"weight":       1,
		"image":        nil,
	}}, "variables.input.image", "product.jpg", []byte("dummy image content"))
	created := tests.Payload(s.T(), create, "createProduct")
	s.productID = tests.Int(s.T(), created, "id")
	s.Require().NotZero(s.productID)
	s.Equal("Graphql Product", tests.Str(s.T(), created, "name"))

	// 2. FindById
	found := s.gql.Do(`
		query Find($input: FindByIdProductInput!) { findByIdProduct(input: $input) { status data { id } } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.productID}})
	s.Equal(s.productID, tests.Int(s.T(), tests.Payload(s.T(), found, "findByIdProduct"), "id"))

	// 3. FindAll
	all := s.gql.Do(`query { findAllProduct(input: {page: 1, pageSize: 10}) { status data { id } } }`, nil)
	s.NotEmpty(tests.Items(s.T(), all, "findAllProduct"))

	// 4. FindByActive
	active := s.gql.Do(`query { findByActiveProduct(input: {page: 1, pageSize: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), active, "findByActiveProduct"))

	// 5. FindByMerchant
	//
	// NOTE: minPrice/maxPrice are optional in the schema but the resolver
	// dereferences them unconditionally, so they must be supplied.
	byMerchant := s.gql.Do(`
		query ByMerchant($input: FindAllProductMerchantInput!) { findByMerchantProduct(input: $input) { status data { id } } }`,
		map[string]interface{}{"input": map[string]interface{}{
			"merchantId": s.merchantID, "page": 1, "pageSize": 10, "minPrice": 0, "maxPrice": 100000,
		}})
	s.NotEmpty(tests.Items(s.T(), byMerchant, "findByMerchantProduct"))

	// 6. FindByCategory
	byCategory := s.gql.Do(`
		query ByCategory($input: FindAllProductCategoryInput!) { findByCategoryProduct(input: $input) { status data { id } } }`,
		map[string]interface{}{"input": map[string]interface{}{
			"categoryName": "Seed Category", "page": 1, "pageSize": 10, "minPrice": 0, "maxPrice": 100000,
		}})
	s.NotEmpty(tests.Items(s.T(), byCategory, "findByCategoryProduct"))

	// 7. Update (multipart; no new image, so image is omitted)
	update := s.gql.DoUpload(`
		mutation Update($input: UpdateProductInput!) {
			updateProduct(input: $input) { status data { id name price } }
		}`, map[string]interface{}{"input": map[string]interface{}{
		"productId":    s.productID,
		"merchantId":   s.merchantID,
		"categoryId":   s.categoryID,
		"name":         "Graphql Product Updated",
		"description":  "Updated Description",
		"price":        2000,
		"countInStock": 20,
		"brand":        "Updated Brand",
		"weight":       2,
	}}, "variables.input.image", "updated.jpg", []byte("updated image content"))
	s.Equal("Graphql Product Updated", tests.Str(s.T(), tests.Payload(s.T(), update, "updateProduct"), "name"))

	// 8. Trash / findByTrashed / restore
	trashed := s.gql.Do(`
		mutation Trash($input: FindByIdProductInput!) { trashedProduct(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.productID}})
	s.Equal("success", tests.Status(s.T(), trashed, "trashedProduct"))

	trashedList := s.gql.Do(`query { findByTrashedProduct(input: {page: 1, pageSize: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), trashedList, "findByTrashedProduct"))

	restored := s.gql.Do(`
		mutation Restore($input: FindByIdProductInput!) { restoreProduct(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.productID}})
	s.Equal("success", tests.Status(s.T(), restored, "restoreProduct"))

	// 9. DeletePermanent
	s.gql.Do(`
		mutation Trash($input: FindByIdProductInput!) { trashedProduct(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.productID}})
	permanent := s.gql.Do(`
		mutation DeletePermanent($input: FindByIdProductInput!) { deleteProductPermanent(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.productID}})
	s.Equal("success", tests.Status(s.T(), permanent, "deleteProductPermanent"))

	// 10. Bulk
	restoreAll := s.gql.Do(`mutation { restoreAllProduct { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), restoreAll, "restoreAllProduct"))
	deleteAll := s.gql.Do(`mutation { deleteAllProductPermanent { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), deleteAll, "deleteAllProductPermanent"))
}

func TestProductGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(ProductGraphqlTestSuite))
}
