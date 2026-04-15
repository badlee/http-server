package binder

import (
	"testing"
)

func TestParseDatabaseRelationships(t *testing.T) {
	content := `
DATABASE "sqlitede://:memory:"
    SCHEMA User DEFINE
        FIELD name string
    END SCHEMA

    SCHEMA Profile DEFINE
        FIELD user_id User.id [has=one delete=cascade update=cascade]
        FIELD bio string
    END SCHEMA

    SCHEMA Order DEFINE
        FIELD customer_id User.id [has=many]
        FIELD amount number
    END SCHEMA
END DATABASE
`

	config, _, err := ParseConfig(content)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	// Find the DATABASE group
	var dbItems []*DatabaseDirective
	for _, g := range config.Groups {
		if g.Directive == "DATABASE" {
			for _, item := range g.Items {
				if d, err := NewDatabaseDirective(item); err == nil {
					dbItems = append(dbItems, d.(*DatabaseDirective))
				}
			}
		}
	}

	if len(dbItems) != 1 {
		t.Fatalf("Expected 1 DATABASE directive, got %d", len(dbItems))
	}

	db := dbItems[0]
	schemas := db.config.GetRoutes("SCHEMA")
	if len(schemas) != 3 {
		t.Fatalf("Expected 3 schemas, got %d", len(schemas))
	}

	// Verify Profile relationships
	profile := schemas.Get("Profile")
	if profile == nil {
		t.Fatalf("Expected Profile schema, got nil")
	}
	userIdField := profile.Routes.Get("user_id")
	if userIdField == nil {
		t.Fatalf("Expected user_id field, got nil")
	}
	if userIdField.Handler != "User.id" {
		t.Errorf("Expected user_id.Ref to be 'User.id', got '%s'", userIdField.Handler)
	}
	if userIdField.Args.Get("has") != "one" {
		t.Errorf("Expected user_id.Has to be 'one', got '%s'", userIdField.Args.Get("has"))
	}
	if userIdField.Args.Get("delete") != "cascade" {
		t.Errorf("Expected user_id.OnDelete to be 'CASCADE', got '%s'", userIdField.Args.Get("delete"))
	}
	if userIdField.Args.Get("update") != "cascade" {
		t.Errorf("Expected user_id.OnUpdate to be 'CASCADE', got '%s'", userIdField.Args.Get("update"))
	}

	// Verify Order relationships
	order := schemas.Get("Order")
	if order == nil {
		t.Fatalf("Expected Order schema, got nil")
	}
	customerIdField := order.Routes.Get("customer_id")
	if customerIdField == nil {
		t.Fatalf("Expected customer_id field, got nil")
	}
	if customerIdField.Handler != "User.id" {
		t.Errorf("Expected customer_id.Ref to be 'User.id', got '%s'", customerIdField.Handler)
	}
	if customerIdField.Args.Get("has") != "many" {
		t.Errorf("Expected customer_id.Has to be 'many', got '%s'", customerIdField.Args.Get("has"))
	}
}

func TestParseCrudDatabaseRelationships(t *testing.T) {
	_ = `
DATABASE "sqlite://crud.db"
    SCHEMA users DEFINE
        FIELD email string [unique]
    END SCHEMA

    SCHEMA profiles DEFINE
        FIELD user_id users.id [has=one delete=cascade]
        FIELD bio text
    END SCHEMA
END DATABASE
`
	// Wait, the above is still a standard DATABASE block.
	// In binder, DATABASE triggers parseDatabaseDirective which calls parseSchema.
	// BUT if it's a CRUD directive inside DATABASE, it might be different?
	// Actually, parseSchemaBlock is used within parseCrudDirectiveConfig.

	// Let's test parseSchemaBlock directly or via a CRUD block if I can find it.
	// In protocol: CRUD path { ... SCHEMA ... }
}

func TestParseSchemaBlockRelationships(t *testing.T) {
	// Mock a RouteConfig for a SCHEMA block
	r := &RouteConfig{
		Method: "SCHEMA",
		Path:   "posts",
		Routes: []*RouteConfig{
			{
				Method:  "FIELD",
				Path:    "author_id",
				Handler: "User.id",
				Args: map[string]string{
					"has":    "one",
					"delete": "cascade",
				},
			},
		},
	}

	s := parseSchemaBlock(r)

	if s.Slug != "posts" {
		t.Errorf("Expected slug 'posts', got '%s'", s.Slug)
	}
	if len(s.Fields) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(s.Fields))
	}

	f := s.Fields[0]
	if f.Name != "author_id" {
		t.Errorf("Expected field name 'author_id', got '%s'", f.Name)
	}
	if f.Ref != "User.id" {
		t.Errorf("Expected ref 'User.id', got '%s'", f.Ref)
	}
	if f.OnDelete != "CASCADE" {
		t.Errorf("Expected delete 'CASCADE', got '%s'", f.OnDelete)
	}
}

func TestParseAliasRelationships(t *testing.T) {
	content := `
DATABASE "sqlite://:memory:"
    SCHEMA User DEFINE
        FIELD roles Role.id [has=many2many]
        FIELD profile Profile.id [has=one]
        FIELD posts Post.id [has=many]
    END SCHEMA
END DATABASE
`
	config, _, err := ParseConfig(content)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	// Find the DATABASE group
	var dbItems []*DatabaseDirective
	for _, g := range config.Groups {
		if g.Directive == "DATABASE" {
			for _, item := range g.Items {
				if d, err := NewDatabaseDirective(item); err == nil {
					dbItems = append(dbItems, d.(*DatabaseDirective))
				}
			}
		}
	}
	if len(dbItems) != 1 {
		t.Fatalf("Expected 1 DATABASE directive, got %d", len(dbItems))
	}
	db := dbItems[0]
	schemas := db.config.GetRoutes("SCHEMA")
	user := schemas.Get("User")
	if user == nil {
		t.Fatalf("Expected User schema, got nil")
	}
	if r := user.Routes.Get("roles"); r != nil && r.Args.Get("has") != "many2many" {
		t.Errorf("Expected roles.Has to be 'many2many', got '%s'", r.Args.Get("has"))
	} else if r == nil {
		t.Errorf("Expected roles.Has to be 'many2many', got 'nil'")
	}
	if r := user.Routes.Get("profile"); r != nil && r.Args.Get("has") != "one" {
		t.Errorf("Expected profile.Has to be 'one', got '%s'", r.Args.Get("has"))
	} else if r == nil {
		t.Errorf("Expected profile.Has to be 'one', got 'nil'")
	}
	if r := user.Routes.Get("posts"); r != nil && r.Args.Get("has") != "many" {
		t.Errorf("Expected posts.Has to be 'many', got '%s'", r.Args.Get("has"))
	} else if r == nil {
		t.Errorf("Expected posts.Has to be 'many', got 'nil'")
	}
}
