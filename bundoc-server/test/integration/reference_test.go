package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// Reference integration tests: schema with x-bundoc-ref, HTTP 409 on FK violation,
// and delete behavior for restrict / set_null / cascade.

func createCollection(projectID string, name string, schema interface{}) (int, error) {
	url := fmt.Sprintf("%s/v1/projects/%s/databases/(default)/collections", serverURL, projectID)
	body := map[string]interface{}{"name": name}
	if schema != nil {
		body["schema"] = schema
	}
	data, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func updateCollectionSchema(projectID, collection string, schema interface{}) (int, error) {
	url := fmt.Sprintf("%s/v1/projects/%s/databases/(default)/collections/%s", serverURL, projectID, collection)
	data, _ := json.Marshal(map[string]interface{}{"schema": schema})
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func createDocumentWithStatus(projectID, collection string, doc map[string]interface{}) (int, []byte, error) {
	url := fmt.Sprintf("%s/v1/projects/%s/databases/(default)/documents/%s", serverURL, projectID, collection)
	data, _ := json.Marshal(doc)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b, nil
}

func patchDocumentWithStatus(projectID, collection, docID string, patch map[string]interface{}) (int, []byte, error) {
	url := fmt.Sprintf("%s/v1/projects/%s/databases/(default)/documents/%s/%s", serverURL, projectID, collection, docID)
	data, _ := json.Marshal(patch)
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b, nil
}

func deleteDocumentWithStatus(projectID, collection, docID string) (int, []byte, error) {
	url := fmt.Sprintf("%s/v1/projects/%s/databases/(default)/documents/%s/%s", serverURL, projectID, collection, docID)
	req, _ := http.NewRequest("DELETE", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b, nil
}

func getDocumentWithStatus(projectID, collection, docID string) (int, map[string]interface{}, error) {
	url := fmt.Sprintf("%s/v1/projects/%s/databases/(default)/documents/%s/%s", serverURL, projectID, collection, docID)
	resp, err := http.Get(url)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	var doc map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&doc)
	return resp.StatusCode, doc, nil
}

func TestReference_CollectionSchemaWithRefExtension(t *testing.T) {
	projectID := "ref-schema-project"
	code, err := createCollection(projectID, "users", nil)
	if err != nil {
		t.Fatalf("create collection users: %v", err)
	}
	if code != http.StatusCreated {
		t.Errorf("create users: expected 201, got %d", code)
	}

	schemaWithRef := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"author_id": map[string]interface{}{
				"type": "string",
				"x-bundoc-ref": map[string]interface{}{
					"collection": "users",
					"field":      "_id",
					"on_delete":  "set_null",
				},
			},
		},
	}
	code, err = createCollection(projectID, "posts", schemaWithRef)
	if err != nil {
		t.Fatalf("create collection posts: %v", err)
	}
	if code != http.StatusCreated {
		t.Errorf("create posts with x-bundoc-ref: expected 201, got %d", code)
	}
}

func TestReference_WriteReturns409WhenTargetMissing(t *testing.T) {
	projectID := "ref-fk-project"
	createCollection(projectID, "users", nil)
	schemaWithRef := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"author_id": map[string]interface{}{
				"type": "string",
				"x-bundoc-ref": map[string]interface{}{"collection": "users", "on_delete": "set_null"},
			},
		},
	}
	createCollection(projectID, "posts", schemaWithRef)

	// Create document referencing non-existent user -> 409
	code, body, _ := createDocumentWithStatus(projectID, "posts", map[string]interface{}{
		"_id":       "p1",
		"author_id": "nonexistent",
	})
	if code != http.StatusConflict {
		t.Errorf("create post with missing target: expected 409, got %d body=%s", code, body)
	}

	// Create user then post (success)
	code, _, _ = createDocumentWithStatus(projectID, "users", map[string]interface{}{"_id": "u1", "name": "Alice"})
	if code != http.StatusCreated {
		t.Fatalf("create user: %d", code)
	}
	code, _, _ = createDocumentWithStatus(projectID, "posts", map[string]interface{}{"_id": "p2", "author_id": "u1"})
	if code != http.StatusCreated {
		t.Fatalf("create post with valid ref: %d", code)
	}

	// Patch to missing target -> 409
	code, _, _ = patchDocumentWithStatus(projectID, "posts", "p2", map[string]interface{}{"author_id": "nonexistent"})
	if code != http.StatusConflict {
		t.Errorf("patch to missing target: expected 409, got %d", code)
	}
}

func TestReference_DeleteRestrictReturns409(t *testing.T) {
	projectID := "ref-restrict-project"
	createCollection(projectID, "users", nil)
	schemaRestrict := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"author_id": map[string]interface{}{
				"type": "string",
				"x-bundoc-ref": map[string]interface{}{"collection": "users", "on_delete": "restrict"},
			},
		},
	}
	createCollection(projectID, "posts", schemaRestrict)

	createDocumentWithStatus(projectID, "users", map[string]interface{}{"_id": "u1", "name": "A"})
	createDocumentWithStatus(projectID, "posts", map[string]interface{}{"_id": "p1", "author_id": "u1"})

	code, _, _ := deleteDocumentWithStatus(projectID, "users", "u1")
	if code != http.StatusConflict {
		t.Errorf("delete user with restrict dependents: expected 409, got %d", code)
	}
}

func TestReference_DeleteSetNullSuccess(t *testing.T) {
	projectID := "ref-setnull-project"
	createCollection(projectID, "users", nil)
	schemaSetNull := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"author_id": map[string]interface{}{
				"type": []interface{}{"string", "null"},
				"x-bundoc-ref": map[string]interface{}{"collection": "users", "on_delete": "set_null"},
			},
			"title": map[string]interface{}{"type": "string"},
		},
	}
	createCollection(projectID, "posts", schemaSetNull)

	createDocumentWithStatus(projectID, "users", map[string]interface{}{"_id": "u1", "name": "A"})
	createDocumentWithStatus(projectID, "posts", map[string]interface{}{"_id": "p1", "author_id": "u1", "title": "Post"})

	code, _, _ := deleteDocumentWithStatus(projectID, "users", "u1")
	if code != http.StatusNoContent {
		t.Fatalf("delete user (set_null): expected 204, got %d", code)
	}

	code, doc, _ := getDocumentWithStatus(projectID, "posts", "p1")
	if code != http.StatusOK {
		t.Fatalf("get post: %d", code)
	}
	if doc["author_id"] != nil {
		t.Errorf("expected author_id null after set_null, got %v", doc["author_id"])
	}
	if doc["title"] != "Post" {
		t.Errorf("expected title unchanged, got %v", doc["title"])
	}
}

func TestReference_DeleteCascadeSuccess(t *testing.T) {
	projectID := "ref-cascade-project"
	createCollection(projectID, "users", nil)
	schemaCascade := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"author_id": map[string]interface{}{
				"type": "string",
				"x-bundoc-ref": map[string]interface{}{"collection": "users", "on_delete": "cascade"},
			},
		},
	}
	createCollection(projectID, "posts", schemaCascade)

	createDocumentWithStatus(projectID, "users", map[string]interface{}{"_id": "u1", "name": "A"})
	createDocumentWithStatus(projectID, "posts", map[string]interface{}{"_id": "p1", "author_id": "u1"})

	code, _, _ := deleteDocumentWithStatus(projectID, "users", "u1")
	if code != http.StatusNoContent {
		t.Fatalf("delete user (cascade): expected 204, got %d", code)
	}

	code, _, _ = getDocumentWithStatus(projectID, "posts", "p1")
	if code != http.StatusNotFound {
		t.Errorf("post should be cascade-deleted: expected 404, got %d", code)
	}
}
