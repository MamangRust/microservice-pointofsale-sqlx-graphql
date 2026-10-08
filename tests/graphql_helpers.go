package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"

	"github.com/MamangRust/microservice-point-of-sale-apigateway/testhelper"
	"github.com/MamangRust/microservice-point-of-sale-pkg/logger"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// GraphQLError is a single entry from a GraphQL error response.
type GraphQLError struct {
	Message string `json:"message"`
}

// graphQLResult is a decoded gqlgen response.
type graphQLResult struct {
	Data   map[string]interface{}
	Errors []GraphQLError
}

// GraphQLClient drives the real apigateway GraphQL handler from integration
// tests. It mirrors the runtime wiring (see service/apigateway/testhelper) so
// the suites exercise the actual resolvers, mappers and caches rather than a
// mock of the gateway layer.
type GraphQLClient struct {
	handler http.Handler
	require *require.Assertions
}

// NewGraphQLClient builds a GraphQL handler backed by the given service
// connections (only the ones a suite needs have to be set) and returns a client
// bound to the supplied testing handle for assertions. The conns map uses the
// same keys as BaseTestSuite.Conns (auth, role, user, category, cashier,
// merchant, order, order-item, product, transaction, stats_reader).
func NewGraphQLClient(t require.TestingT, conns map[string]*grpc.ClientConn, redisClient *redis.Client, log logger.LoggerInterface) *GraphQLClient {
	sc := testhelper.SetupServiceConnections(conns)
	handler := testhelper.NewTestHandler(sc, redisClient, log)
	return &GraphQLClient{
		handler: handler,
		require: require.New(t),
	}
}

// WithUser returns a client whose requests carry userID in the context, which
// is what AuthMiddleware does for authenticated traffic. Resolvers that read
// the caller identity from the context need this.
func (c *GraphQLClient) WithUser(userID int) *GraphQLClient {
	return &GraphQLClient{
		handler: testhelper.WithUser(c.handler, userID),
		require: c.require,
	}
}

// WithUserAndToken returns a client whose requests carry both the caller id and
// the access token, as AuthMiddleware does. GetMe needs the token too.
func (c *GraphQLClient) WithUserAndToken(userID int, token string) *GraphQLClient {
	return &GraphQLClient{
		handler: testhelper.WithUserAndToken(c.handler, userID, token),
		require: c.require,
	}
}

// Do executes a query/mutation and fails the test on transport or GraphQL
// errors, returning the data object.
func (c *GraphQLClient) Do(query string, vars map[string]interface{}) map[string]interface{} {
	data, errs := c.do(query, vars)
	c.require.Emptyf(errs, "unexpected graphql errors: %+v", errs)
	return data
}

// DoRaw executes a query/mutation and returns both data and errors without
// failing, for suites that assert on the error path.
func (c *GraphQLClient) DoRaw(query string, vars map[string]interface{}) (map[string]interface{}, []GraphQLError) {
	return c.do(query, vars)
}

func (c *GraphQLClient) do(query string, vars map[string]interface{}) (map[string]interface{}, []GraphQLError) {
	payload := map[string]interface{}{"query": query}
	if vars != nil {
		payload["variables"] = vars
	}
	body, err := json.Marshal(payload)
	c.require.NoError(err)

	req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)

	return c.decode(rec)
}

// DoUpload executes a mutation that carries a file upload (the gqlgen
// multipart request spec: operations + map + file parts). path is a JSON path
// into the variables map that should receive the file, e.g.
// "variables.input.image".
func (c *GraphQLClient) DoUpload(query string, vars map[string]interface{}, path, filename string, content []byte) map[string]interface{} {
	operations, err := json.Marshal(map[string]interface{}{"query": query, "variables": vars})
	c.require.NoError(err)
	mapping, err := json.Marshal(map[string][]string{"0": {path}})
	c.require.NoError(err)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	ow, err := w.CreateFormField("operations")
	c.require.NoError(err)
	_, err = ow.Write(operations)
	c.require.NoError(err)
	mw, err := w.CreateFormField("map")
	c.require.NoError(err)
	_, err = mw.Write(mapping)
	c.require.NoError(err)
	fw, err := w.CreateFormFile("0", filename)
	c.require.NoError(err)
	_, err = fw.Write(content)
	c.require.NoError(err)
	c.require.NoError(w.Close())

	req := httptest.NewRequest(http.MethodPost, "/query", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)

	data, errs := c.decode(rec)
	c.require.Emptyf(errs, "unexpected graphql errors: %+v", errs)
	return data
}

func (c *GraphQLClient) decode(rec *httptest.ResponseRecorder) (map[string]interface{}, []GraphQLError) {
	raw, err := io.ReadAll(rec.Result().Body)
	c.require.NoError(err)

	var resp graphQLResult
	c.require.NoErrorf(json.Unmarshal(raw, &resp), "response body: %s", string(raw))
	return resp.Data, resp.Errors
}

// Obj extracts data[field] as an object.
func Obj(t require.TestingT, data map[string]interface{}, field string) map[string]interface{} {
	v, ok := data[field].(map[string]interface{})
	require.New(t).Truef(ok, "expected %q to be an object, got %T", field, data[field])
	return v
}

// Payload extracts data[field].data (the ApiResponse shape).
func Payload(t require.TestingT, data map[string]interface{}, field string) map[string]interface{} {
	return Obj(t, Obj(t, data, field), "data")
}

// Items extracts data[field].data as a list.
func Items(t require.TestingT, data map[string]interface{}, field string) []interface{} {
	obj := Obj(t, data, field)
	list, ok := obj["data"].([]interface{})
	require.New(t).Truef(ok, "expected %s.data to be a list, got %T", field, obj["data"])
	return list
}

// Status returns data[field].status.
func Status(t require.TestingT, data map[string]interface{}, field string) string {
	return Str(t, Obj(t, data, field), "status")
}

// Int reads a numeric field from a JSON object.
func Int(t require.TestingT, m map[string]interface{}, key string) int {
	r := require.New(t)
	v, ok := m[key].(float64)
	r.Truef(ok, "expected %q to be a number, got %T", key, m[key])
	return int(v)
}

// Str reads a string field from a JSON object.
func Str(t require.TestingT, m map[string]interface{}, key string) string {
	r := require.New(t)
	v, ok := m[key].(string)
	r.Truef(ok, "expected %q to be a string, got %T", key, m[key])
	return v
}
