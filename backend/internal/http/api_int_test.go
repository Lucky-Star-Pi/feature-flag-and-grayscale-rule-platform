package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"featureflag/internal/db"
	"featureflag/internal/migrateutil"
	"featureflag/internal/model"
	"featureflag/internal/service"
	"featureflag/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupAPI(t *testing.T) (*gin.Engine, *db.DB) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL 未设置，跳过集成测试")
	}
	mig := migrationsFileURL(t)
	_, _, err := migrateutil.Up(mig, dsn)
	require.NoError(t, err)

	database, err := db.Open(dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })

	svc := service.New(database)
	return NewRouter(svc), database
}

func migrationsFileURL(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	p := filepath.Clean(filepath.Join(wd, "..", "..", "migrations"))
	abs, err := filepath.Abs(p)
	require.NoError(t, err)
	return "file://" + filepath.ToSlash(abs)
}

func uniqueKey(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

type errResp struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func doJSON(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDuplicateKeySameEnv_409_NoHistory(t *testing.T) {
	r, database := setupAPI(t)
	key := uniqueKey("dup")
	w1 := doJSON(t, r, http.MethodPost, "/api/v1/flags", map[string]any{
		"name": "A", "key": key, "environment": "development", "defaultValue": false,
	})
	require.Equal(t, http.StatusCreated, w1.Code, w1.Body.String())

	var created struct {
		Flag model.Flag `json:"flag"`
	}
	require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &created))

	before, err := store.ListHistory(context.Background(), database.SQL, created.Flag.ID)
	require.NoError(t, err)

	w2 := doJSON(t, r, http.MethodPost, "/api/v1/flags", map[string]any{
		"name": "B", "key": key, "environment": "development", "defaultValue": true,
	})
	require.Equal(t, http.StatusConflict, w2.Code, w2.Body.String())
	var er errResp
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &er))
	require.Equal(t, "KEY_CONFLICT", er.Error.Code)

	after, err := store.ListHistory(context.Background(), database.SQL, created.Flag.ID)
	require.NoError(t, err)
	require.Equal(t, len(before), len(after), "重复 Key 失败不得写入 history")
}

func TestSameKeyDifferentEnv_OK(t *testing.T) {
	r, _ := setupAPI(t)
	key := uniqueKey("cross")
	w1 := doJSON(t, r, http.MethodPost, "/api/v1/flags", map[string]any{
		"name": "Dev", "key": key, "environment": "development", "defaultValue": false,
	})
	require.Equal(t, http.StatusCreated, w1.Code, w1.Body.String())
	w2 := doJSON(t, r, http.MethodPost, "/api/v1/flags", map[string]any{
		"name": "Prod", "key": key, "environment": "production", "defaultValue": false,
	})
	require.Equal(t, http.StatusCreated, w2.Code, w2.Body.String())
}

func TestCreateFlag_WritesHistoryInSameTx(t *testing.T) {
	r, _ := setupAPI(t)
	key := uniqueKey("hist")
	w := doJSON(t, r, http.MethodPost, "/api/v1/flags", map[string]any{
		"name": "H", "key": key, "environment": "staging", "defaultValue": true, "enabled": true,
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created struct {
		Flag model.Flag `json:"flag"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.NotZero(t, created.Flag.ID)

	hw := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/flags/%d/history", created.Flag.ID), nil)
	require.Equal(t, http.StatusOK, hw.Code)
	var hist struct {
		Items []model.History `json:"items"`
	}
	require.NoError(t, json.Unmarshal(hw.Body.Bytes(), &hist))
	require.NotEmpty(t, hist.Items)
	require.Equal(t, model.OpCreateFlag, hist.Items[0].OperationType)
	require.Equal(t, model.ActorLocalAdmin, hist.Items[0].Operator)
	require.Contains(t, hist.Items[0].Summary, key)
}

func TestDuplicatePriority_400_NoHistoryResidue(t *testing.T) {
	r, database := setupAPI(t)
	key := uniqueKey("pri")
	w := doJSON(t, r, http.MethodPost, "/api/v1/flags", map[string]any{
		"name": "P", "key": key, "environment": "development", "defaultValue": false,
	})
	require.Equal(t, http.StatusCreated, w.Code)
	var created struct {
		Flag model.Flag `json:"flag"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	flagID := created.Flag.ID

	ruleBody := map[string]any{
		"attribute": "country", "operator": "equals", "expectedValue": "CN",
		"returnValue": true, "priority": 0,
	}
	w1 := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/v1/flags/%d/rules", flagID), ruleBody)
	require.Equal(t, http.StatusCreated, w1.Code, w1.Body.String())

	before, err := store.ListHistory(context.Background(), database.SQL, flagID)
	require.NoError(t, err)

	ruleBody["attribute"] = "plan"
	w2 := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/v1/flags/%d/rules", flagID), ruleBody)
	require.Equal(t, http.StatusBadRequest, w2.Code, w2.Body.String())
	var er errResp
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &er))
	require.Equal(t, "PRIORITY_CONFLICT", er.Error.Code)

	after, err := store.ListHistory(context.Background(), database.SQL, flagID)
	require.NoError(t, err)
	require.Equal(t, len(before), len(after))

	rules, err := store.ListRules(context.Background(), database.SQL, flagID)
	require.NoError(t, err)
	require.Len(t, rules, 1)
}

func TestUpdateEnableHistorySummary(t *testing.T) {
	r, _ := setupAPI(t)
	key := uniqueKey("upd")
	w := doJSON(t, r, http.MethodPost, "/api/v1/flags", map[string]any{
		"name": "OldName", "key": key, "environment": "development", "defaultValue": false,
	})
	require.Equal(t, http.StatusCreated, w.Code)
	var created struct {
		Flag model.Flag `json:"flag"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	id := created.Flag.ID

	pw := doJSON(t, r, http.MethodPatch, fmt.Sprintf("/api/v1/flags/%d", id), map[string]any{
		"name": "NewName", "defaultValue": true,
	})
	require.Equal(t, http.StatusOK, pw.Code, pw.Body.String())

	dw := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/v1/flags/%d/disable", id), nil)
	require.Equal(t, http.StatusOK, dw.Code, dw.Body.String())

	ew := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/v1/flags/%d/enable", id), nil)
	require.Equal(t, http.StatusOK, ew.Code, ew.Body.String())

	hw := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/flags/%d/history", id), nil)
	require.Equal(t, http.StatusOK, hw.Code)
	var hist struct {
		Items []model.History `json:"items"`
	}
	require.NoError(t, json.Unmarshal(hw.Body.Bytes(), &hist))
	types := map[string]string{}
	for _, h := range hist.Items {
		types[h.OperationType] = h.Summary
	}
	require.Contains(t, types, model.OpUpdateFlag)
	require.Contains(t, types[model.OpUpdateFlag], "OldName → NewName")
	require.Contains(t, types[model.OpUpdateFlag], "false → true")
	require.Contains(t, types, model.OpDisableFlag)
	require.Contains(t, types[model.OpDisableFlag], "enabled:")
	require.Contains(t, types, model.OpEnableFlag)
}

func TestNotFound(t *testing.T) {
	r, _ := setupAPI(t)
	missing := int64(9_999_999)

	w := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/flags/%d", missing), nil)
	require.Equal(t, http.StatusNotFound, w.Code)
	var er errResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &er))
	require.Equal(t, "NOT_FOUND", er.Error.Code)

	w = doJSON(t, r, http.MethodPatch, fmt.Sprintf("/api/v1/flags/%d", missing), map[string]any{
		"name": "x", "defaultValue": false,
	})
	require.Equal(t, http.StatusNotFound, w.Code)

	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/v1/flags/%d/enable", missing), nil)
	require.Equal(t, http.StatusNotFound, w.Code)

	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/v1/flags/%d/rules", missing), map[string]any{
		"attribute": "country", "operator": "equals", "expectedValue": "CN",
		"returnValue": true, "priority": 0,
	})
	require.Equal(t, http.StatusNotFound, w.Code)

	w = doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/flags/%d/rules/1", missing), nil)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestInvalidEnvironment_400(t *testing.T) {
	r, _ := setupAPI(t)
	w := doJSON(t, r, http.MethodPost, "/api/v1/flags", map[string]any{
		"name": "X", "key": uniqueKey("badenv"), "environment": "prod", "defaultValue": false,
	})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	var er errResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &er))
	require.Equal(t, "INVALID_INPUT", er.Error.Code)
}

func TestRulesCRUD_AndDetailOrder(t *testing.T) {
	r, _ := setupAPI(t)
	w := doJSON(t, r, http.MethodPost, "/api/v1/flags", map[string]any{
		"name": "R", "key": uniqueKey("rules"), "environment": "development", "defaultValue": false,
	})
	require.Equal(t, http.StatusCreated, w.Code)
	var created struct {
		Flag model.Flag `json:"flag"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	id := created.Flag.ID

	w10 := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/v1/flags/%d/rules", id), map[string]any{
		"attribute": "plan", "operator": "in", "expectedValue": `["pro"]`,
		"returnValue": true, "priority": 10,
	})
	require.Equal(t, http.StatusCreated, w10.Code, w10.Body.String())

	w0 := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/v1/flags/%d/rules", id), map[string]any{
		"attribute": "country", "operator": "equals", "expectedValue": "CN",
		"returnValue": true, "priority": 0,
	})
	require.Equal(t, http.StatusCreated, w0.Code)

	var rule0 struct {
		Rule model.Rule `json:"rule"`
	}
	require.NoError(t, json.Unmarshal(w0.Body.Bytes(), &rule0))

	uw := doJSON(t, r, http.MethodPatch, fmt.Sprintf("/api/v1/flags/%d/rules/%d", id, rule0.Rule.ID), map[string]any{
		"attribute": "country", "operator": "equals", "expectedValue": "US",
		"returnValue": false, "priority": 0,
	})
	require.Equal(t, http.StatusOK, uw.Code, uw.Body.String())

	detail := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/flags/%d", id), nil)
	require.Equal(t, http.StatusOK, detail.Code)
	var d service.FlagDetail
	require.NoError(t, json.Unmarshal(detail.Body.Bytes(), &d))
	require.Len(t, d.Rules, 2)
	require.Equal(t, 0, d.Rules[0].Priority)
	require.Equal(t, 10, d.Rules[1].Priority)
	require.Equal(t, "US", d.Rules[0].ExpectedValue)

	del := doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/flags/%d/rules/%d", id, rule0.Rule.ID), nil)
	require.Equal(t, http.StatusNoContent, del.Code)

	hw := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/flags/%d/history", id), nil)
	var hist struct {
		Items []model.History `json:"items"`
	}
	require.NoError(t, json.Unmarshal(hw.Body.Bytes(), &hist))
	ops := map[string]bool{}
	for _, h := range hist.Items {
		ops[h.OperationType] = true
	}
	require.True(t, ops[model.OpCreateRule])
	require.True(t, ops[model.OpUpdateRule])
	require.True(t, ops[model.OpDeleteRule])
}
