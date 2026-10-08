package order_item_test

import (
	"testing"

	tests "github.com/MamangRust/microservice-point-of-sale-test"
	"github.com/stretchr/testify/suite"
)

type OrderItemGraphqlTestSuite struct {
	tests.BaseTestSuite
	gql         *tests.GraphQLClient
	userID      int
	categoryID  int
	merchantID  int
	productID   int
	orderID     int
	orderItemID int
}

func (s *OrderItemGraphqlTestSuite) SetupSuite() {
	s.BaseTestSuite.SetupSuite()

	// SetupOrderService chains cashier, merchant, product and order-item.
	s.SetupOrderService()

	s.userID = s.SeedUser(s.Ctx)
	s.categoryID = s.SeedCategory(s.Ctx)
	s.merchantID = s.SeedMerchant(s.Ctx, s.userID)
	s.productID = s.SeedProduct(s.Ctx, s.merchantID, s.categoryID)
	s.orderID = s.SeedOrder(s.Ctx, s.userID, s.merchantID, s.productID)

	s.gql = tests.NewGraphQLClient(s.T(), s.Conns, s.RedisClient(), s.Log)
}

func (s *OrderItemGraphqlTestSuite) TestOrderItemGraphqlLifecycle() {
	// The order-item service is read-only, so seed an extra item directly to
	// make sure findByOrder/findAll have something to return.
	s.orderItemID = s.SeedOrderItem(s.Ctx, s.orderID, s.productID)
	s.Require().NotZero(s.orderItemID)

	// 1. FindOrderItemByOrder (input.id is the order id)
	byOrder := s.gql.Do(`
		query ByOrder($input: FindByIdOrderItemInput!) {
			findOrderItemByOrder(input: $input) { status message data { id order_id product_id } }
		}`, map[string]interface{}{"input": map[string]interface{}{"id": s.orderID}})
	s.NotEmpty(tests.Items(s.T(), byOrder, "findOrderItemByOrder"))

	// 2. FindAll
	all := s.gql.Do(`query { findAllOrderItem(input: {page: 1, page_size: 10}) { status data { id } } }`, nil)
	s.NotEmpty(tests.Items(s.T(), all, "findAllOrderItem"))

	// 3. FindByActive
	active := s.gql.Do(`query { findByActiveOrderItem(input: {page: 1, page_size: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), active, "findByActiveOrderItem"))

	// 4. Trash directly in DB (the service exposes no write path), then
	// FindByTrashed.
	_, err := s.SQLxDB().ExecContext(s.Ctx,
		`UPDATE order_items SET deleted_at = NOW() WHERE order_item_id = $1`, s.orderItemID)
	s.Require().NoError(err)

	trashed := s.gql.Do(`query { findByTrashedOrderItem(input: {page: 1, page_size: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), trashed, "findByTrashedOrderItem"))
}

func TestOrderItemGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(OrderItemGraphqlTestSuite))
}
