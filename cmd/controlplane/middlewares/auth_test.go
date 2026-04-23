// SPDX-License-Identifier: GPL-3.0-or-later
package middlewares

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/lopster568/phantomDNS/internal/storage/models"
	"github.com/lopster568/phantomDNS/internal/storage/repositories"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Token{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func buildRouter(users repositories.UserRepository, tokens repositories.TokenRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Auth(users, tokens))
	r.GET("/api/v1/auth/login", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/api/v1/policies", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/api/v1/policies",
		RequireRole(models.RoleOperator),
		func(c *gin.Context) { c.Status(http.StatusCreated) })
	r.POST("/api/v1/users",
		RequireRole(models.RoleAdmin),
		func(c *gin.Context) { c.Status(http.StatusCreated) })
	return r
}

func do(r http.Handler, method, path, bearer string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func seedUserWithToken(t *testing.T, db *gorm.DB, role string) (*models.User, string) {
	t.Helper()
	ur := repositories.NewUserRepo(db)
	tr := repositories.NewTokenRepo(db)
	u, err := ur.Create("u"+role+"@x.com", "hash", role)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	plaintext, _, err := tr.CreateForUser(u.ID, "test", time.Hour)
	if err != nil {
		t.Fatalf("seed token: %v", err)
	}
	return u, plaintext
}

func TestAuth_ExemptPathsBypassEverything(t *testing.T) {
	db := openAuthTestDB(t)
	r := buildRouter(repositories.NewUserRepo(db), repositories.NewTokenRepo(db))

	// No users exist, no token, but login path is exempt.
	rec := do(r, "GET", "/api/v1/auth/login", "")
	if rec.Code != http.StatusOK {
		t.Errorf("exempt path returned %d, want 200", rec.Code)
	}
}

func TestAuth_NoUsers_BlocksProtectedRoutes(t *testing.T) {
	db := openAuthTestDB(t)
	r := buildRouter(repositories.NewUserRepo(db), repositories.NewTokenRepo(db))

	// Zero users → any protected route should 403 with a setup hint.
	rec := do(r, "GET", "/api/v1/policies", "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("no-users pre-setup: got %d, want 403", rec.Code)
	}
}

func TestAuth_MissingBearer_401(t *testing.T) {
	db := openAuthTestDB(t)
	seedUserWithToken(t, db, models.RoleAdmin)
	r := buildRouter(repositories.NewUserRepo(db), repositories.NewTokenRepo(db))

	rec := do(r, "GET", "/api/v1/policies", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("missing bearer: got %d, want 401", rec.Code)
	}
}

func TestAuth_InvalidToken_401(t *testing.T) {
	db := openAuthTestDB(t)
	seedUserWithToken(t, db, models.RoleAdmin)
	r := buildRouter(repositories.NewUserRepo(db), repositories.NewTokenRepo(db))

	rec := do(r, "GET", "/api/v1/policies", "nope-this-token-does-not-exist")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("invalid bearer: got %d, want 401", rec.Code)
	}
}

func TestAuth_ValidToken_AllowsRead(t *testing.T) {
	db := openAuthTestDB(t)
	_, token := seedUserWithToken(t, db, models.RoleReadOnly)
	r := buildRouter(repositories.NewUserRepo(db), repositories.NewTokenRepo(db))

	rec := do(r, "GET", "/api/v1/policies", token)
	if rec.Code != http.StatusOK {
		t.Errorf("read_only on GET: got %d, want 200", rec.Code)
	}
}

func TestRequireRole_ReadOnlyBlockedFromWrite(t *testing.T) {
	db := openAuthTestDB(t)
	_, token := seedUserWithToken(t, db, models.RoleReadOnly)
	r := buildRouter(repositories.NewUserRepo(db), repositories.NewTokenRepo(db))

	rec := do(r, "POST", "/api/v1/policies", token)
	if rec.Code != http.StatusForbidden {
		t.Errorf("read_only on POST: got %d, want 403", rec.Code)
	}
}

func TestRequireRole_OperatorAllowedOnOperatorWrite(t *testing.T) {
	db := openAuthTestDB(t)
	_, token := seedUserWithToken(t, db, models.RoleOperator)
	r := buildRouter(repositories.NewUserRepo(db), repositories.NewTokenRepo(db))

	rec := do(r, "POST", "/api/v1/policies", token)
	if rec.Code != http.StatusCreated {
		t.Errorf("operator on POST /policies: got %d, want 201", rec.Code)
	}
}

func TestRequireRole_OperatorBlockedFromAdminEndpoint(t *testing.T) {
	db := openAuthTestDB(t)
	_, token := seedUserWithToken(t, db, models.RoleOperator)
	r := buildRouter(repositories.NewUserRepo(db), repositories.NewTokenRepo(db))

	rec := do(r, "POST", "/api/v1/users", token)
	if rec.Code != http.StatusForbidden {
		t.Errorf("operator on POST /users: got %d, want 403", rec.Code)
	}
}

func TestRequireRole_AdminBypassesEverything(t *testing.T) {
	db := openAuthTestDB(t)
	_, token := seedUserWithToken(t, db, models.RoleAdmin)
	r := buildRouter(repositories.NewUserRepo(db), repositories.NewTokenRepo(db))

	// Admin hits an operator-tagged endpoint
	rec := do(r, "POST", "/api/v1/policies", token)
	if rec.Code != http.StatusCreated {
		t.Errorf("admin on operator endpoint: got %d", rec.Code)
	}
	// And an admin-tagged endpoint
	rec = do(r, "POST", "/api/v1/users", token)
	if rec.Code != http.StatusCreated {
		t.Errorf("admin on admin endpoint: got %d", rec.Code)
	}
}

func TestAuth_DisabledUserTreatedAsInvalid(t *testing.T) {
	db := openAuthTestDB(t)
	u, token := seedUserWithToken(t, db, models.RoleAdmin)
	if err := repositories.NewUserRepo(db).SetDisabled(u.ID, true); err != nil {
		t.Fatalf("disable: %v", err)
	}
	r := buildRouter(repositories.NewUserRepo(db), repositories.NewTokenRepo(db))

	rec := do(r, "GET", "/api/v1/policies", token)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("disabled user: got %d, want 401", rec.Code)
	}
}

func TestUserFromContext_NoUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if _, ok := UserFromContext(c); ok {
		t.Error("empty context should yield ok=false")
	}
}

// errOnCountUserRepo forces UserRepository.Count to fail, exercising the
// 500 path in Auth.
type errOnCountUserRepo struct{ repositories.UserRepository }

func (errOnCountUserRepo) Count() (int64, error) { return 0, errors.New("boom") }

func TestAuth_UserRepoFailure_500(t *testing.T) {
	db := openAuthTestDB(t)
	real := repositories.NewUserRepo(db)
	r := buildRouter(errOnCountUserRepo{real}, repositories.NewTokenRepo(db))

	rec := do(r, "GET", "/api/v1/policies", "")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("repo failure: got %d, want 500", rec.Code)
	}
}
