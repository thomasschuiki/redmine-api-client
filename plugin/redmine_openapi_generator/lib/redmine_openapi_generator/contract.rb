module RedmineOpenapiGenerator
  # Contract describes the Redmine REST API payload shapes: request bodies and
  # response objects for every resource the CLI consumes, plus shared component
  # schemas. It is the source of truth for SpecBuilder's schema emission and
  # replaces the previous DB-column introspection.
  #
  # The shapes here are derived from the working hand-written types in
  # internal/client/*.go (themselves validated against a live Redmine instance)
  # plus well-known Redmine REST API conventions.
  #
  # Schema fragments are OpenAPI 3.0 schema objects. A $ref is written as the bare
  # component name (e.g. "IDName"); SpecBuilder resolves it to
  # "#/components/schemas/IDName".
  class Contract
    # Shared component schemas referenced across resources.
    SHARED = {
      'IDName' => {
        'type' => 'object',
        'required' => %w[id name],
        'properties' => {
          'id' => { 'type' => 'integer' },
          'name' => { 'type' => 'string' }
        }
      },
      'IssueRef' => {
        'type' => 'object',
        'required' => %w[id],
        'properties' => {
          'id' => { 'type' => 'integer' }
        }
      },
      'Journal' => {
        'type' => 'object',
        'required' => %w[id],
        'properties' => {
          'id' => { 'type' => 'integer' },
          'notes' => { 'type' => 'string' },
          'private_notes' => { 'type' => 'boolean' },
          'created_on' => { 'type' => 'string', 'format' => 'date-time' },
          'user' => { '$ref' => 'IDName' },
          'details' => {
            'type' => 'array',
            'items' => { '$ref' => 'JournalDetail' }
          }
        }
      },
      'JournalDetail' => {
        'type' => 'object',
        'properties' => {
          'property' => { 'type' => 'string' },
          'name' => { 'type' => 'string' },
          'old_value' => { 'type' => 'string' },
          'new_value' => { 'type' => 'string' }
        }
      },
      'CustomField' => {
        'type' => 'object',
        'required' => %w[id name],
        'properties' => {
          'id' => { 'type' => 'integer' },
          'name' => { 'type' => 'string' },
          'value' => {},
          'multiple' => { 'type' => 'boolean' }
        }
      },
      'CustomFieldValue' => {
        'type' => 'object',
        'required' => %w[id],
        'properties' => {
          'id' => { 'type' => 'integer' },
          'value' => { 'type' => 'string' },
          'values' => {
            'type' => 'array',
            'items' => { 'type' => 'string' }
          }
        }
      },
      'Error' => {
        'type' => 'object',
        'properties' => {
          'errors' => {
            'type' => 'array',
            'items' => { 'type' => 'string' }
          }
        }
      }
    }.freeze

    # Per-resource contract. Each entry:
    #
    #   'request'  => { 'wrapper' => 'issue', 'required' => [...], 'properties' => {...} }
    #                 Flat fields sent in create bodies, wrapped by 'wrapper'.
    #                 nil for read-only resources (trackers, statuses, priorities, ...).
    #   'update'   => same shape as 'request'; emitted as {Name}Update when present.
    #                 nil (or absent) when the resource has no update operation.
    #   'single'   => { 'wrapper' => 'issue', 'required' => [...], 'properties' => {...} }
    #                 Flat response object, wrapped by 'wrapper' in single responses.
    #   'list'     => { 'wrapper' => 'issues', 'envelope' => true, 'required' => [...], 'properties' => {...} }
    #                 Response wrapper. When 'envelope' is true, total_count/offset/limit
    #                 are added alongside the items array.
    #   'path_key' => the URL path parameter name for the resource ('id' or 'identifier').
    #
    # Property values are OpenAPI schema fragments; '$ref' carries a bare component
    # name (see SHARED). For list schemas, the special property '__items__' expands
    # to the array of the resource's 'single' schema.
    RESOURCES = {
      'Issue' => {
        'request' => {
          'wrapper' => 'issue',
          'required' => %w[subject],
          'properties' => {
            'subject' => { 'type' => 'string' },
            'project_id' => { 'type' => 'integer' },
            'description' => { 'type' => 'string' },
            'tracker_id' => { 'type' => 'integer' },
            'status_id' => { 'type' => 'integer' },
            'priority_id' => { 'type' => 'integer' },
            'category_id' => { 'type' => 'integer' },
            'fixed_version_id' => { 'type' => 'integer' },
            'assigned_to_id' => { 'type' => 'integer' },
            'parent_id' => { 'type' => 'integer' },
            'start_date' => { 'type' => 'string', 'format' => 'date' },
            'due_date' => { 'type' => 'string', 'format' => 'date' },
            'estimated_hours' => { 'type' => 'number' },
            'watcher_user_ids' => { 'type' => 'array', 'items' => { 'type' => 'integer' } },
            'custom_fields' => { 'type' => 'array', 'items' => { '$ref' => 'CustomFieldValue' } }
          }
        },
        'update' => {
          'wrapper' => 'issue',
          'properties' => {
            'subject' => { 'type' => 'string' },
            'description' => { 'type' => 'string' },
            'start_date' => { 'type' => 'string', 'format' => 'date' },
            'due_date' => { 'type' => 'string', 'format' => 'date' },
            'estimated_hours' => { 'type' => 'number' },
            'tracker_id' => { 'type' => 'integer' },
            'status_id' => { 'type' => 'integer' },
            'priority_id' => { 'type' => 'integer' },
            'assigned_to_id' => { 'type' => 'integer' },
            'done_ratio' => { 'type' => 'integer' },
            'fixed_version_id' => { 'type' => 'integer' },
            'category_id' => { 'type' => 'integer' },
            'notes' => { 'type' => 'string' },
            'custom_fields' => { 'type' => 'array', 'items' => { '$ref' => 'CustomFieldValue' } }
          }
        },
        'single' => {
          'wrapper' => 'issue',
          'required' => %w[id subject],
          'properties' => {
            'id' => { 'type' => 'integer' },
            'subject' => { 'type' => 'string' },
            'description' => { 'type' => 'string' },
            'done_ratio' => { 'type' => 'integer' },
            'is_private' => { 'type' => 'boolean' },
            'start_date' => { 'type' => 'string', 'format' => 'date' },
            'due_date' => { 'type' => 'string', 'format' => 'date' },
            'created_on' => { 'type' => 'string', 'format' => 'date-time' },
            'updated_on' => { 'type' => 'string', 'format' => 'date-time' },
            'closed_on' => { 'type' => 'string', 'format' => 'date-time' },
            'parent' => { '$ref' => 'IssueRef' },
            'project' => { '$ref' => 'IDName' },
            'tracker' => { '$ref' => 'IDName' },
            'status' => { '$ref' => 'IDName' },
            'priority' => { '$ref' => 'IDName' },
            'author' => { '$ref' => 'IDName' },
            'assigned_to' => { '$ref' => 'IDName' },
            'custom_fields' => { 'type' => 'array', 'items' => { '$ref' => 'CustomField' } },
            'journals' => { 'type' => 'array', 'items' => { '$ref' => 'Journal' } },
            'relations' => { 'type' => 'array', 'items' => { '$ref' => 'Relation' } }
          }
        },
        'list' => {
          'wrapper' => 'issues',
          'envelope' => true,
          'properties' => { '__items__' => true }
        },
        'path_key' => 'id'
      },
      'Project' => {
        'request' => {
          'wrapper' => 'project',
          'required' => %w[name identifier],
          'properties' => {
            'name' => { 'type' => 'string' },
            'identifier' => { 'type' => 'string' },
            'description' => { 'type' => 'string' },
            'homepage' => { 'type' => 'string' },
            'is_public' => { 'type' => 'boolean' },
            'parent_id' => { 'type' => 'integer' },
            'inherit_members' => { 'type' => 'boolean' },
            'default_version_id' => { 'type' => 'integer' },
            'default_assigned_to_id' => { 'type' => 'integer' }
          }
        },
        'single' => {
          'wrapper' => 'project',
          'required' => %w[id name identifier],
          'properties' => {
            'id' => { 'type' => 'integer' },
            'name' => { 'type' => 'string' },
            'identifier' => { 'type' => 'string' },
            'description' => { 'type' => 'string' },
            'homepage' => { 'type' => 'string' },
            'is_public' => { 'type' => 'boolean' },
            'status' => { 'type' => 'integer' },
            'created_on' => { 'type' => 'string', 'format' => 'date-time' },
            'updated_on' => { 'type' => 'string', 'format' => 'date-time' },
            'parent' => { '$ref' => 'IDName' },
            'custom_fields' => { 'type' => 'array', 'items' => { '$ref' => 'CustomField' } }
          }
        },
        'list' => {
          'wrapper' => 'projects',
          'envelope' => true,
          'properties' => { '__items__' => true }
        },
        'path_key' => 'identifier'
      },
      'User' => {
        'request' => nil,
        'single' => {
          'wrapper' => 'user',
          'required' => %w[id],
          'properties' => {
            'id' => { 'type' => 'integer' },
            'login' => { 'type' => 'string' },
            'firstname' => { 'type' => 'string' },
            'lastname' => { 'type' => 'string' },
            'mail' => { 'type' => 'string' },
            'admin' => { 'type' => 'boolean' },
            'status' => { 'type' => 'integer' },
            'created_on' => { 'type' => 'string', 'format' => 'date-time' },
            'updated_on' => { 'type' => 'string', 'format' => 'date-time' },
            'last_login_on' => { 'type' => 'string', 'format' => 'date-time' },
            'custom_fields' => { 'type' => 'array', 'items' => { '$ref' => 'CustomField' } }
          }
        },
        'list' => {
          'wrapper' => 'users',
          'envelope' => true,
          'properties' => { '__items__' => true }
        },
        'path_key' => 'id'
      },
      'TimeEntry' => {
        'request' => {
          'wrapper' => 'time_entry',
          'required' => %w[hours],
          'properties' => {
            'hours' => { 'type' => 'number' },
            'issue_id' => { 'type' => 'integer' },
            'project_id' => { 'type' => 'integer' },
            'activity_id' => { 'type' => 'integer' },
            'comments' => { 'type' => 'string' },
            'spent_on' => { 'type' => 'string', 'format' => 'date' }
          }
        },
        'single' => {
          'wrapper' => 'time_entry',
          'required' => %w[id hours],
          'properties' => {
            'id' => { 'type' => 'integer' },
            'hours' => { 'type' => 'number' },
            'comments' => { 'type' => 'string' },
            'spent_on' => { 'type' => 'string', 'format' => 'date' },
            'created_on' => { 'type' => 'string', 'format' => 'date-time' },
            'updated_on' => { 'type' => 'string', 'format' => 'date-time' },
            'project' => { '$ref' => 'IDName' },
            'issue' => { '$ref' => 'IDName' },
            'user' => { '$ref' => 'IDName' },
            'activity' => { '$ref' => 'IDName' }
          }
        },
        'list' => {
          'wrapper' => 'time_entries',
          'envelope' => true,
          'properties' => { '__items__' => true }
        },
        'path_key' => 'id'
      },
      'WikiPage' => {
        'request' => nil,
        'single' => {
          'wrapper' => 'wiki_page',
          'required' => %w[id title],
          'properties' => {
            'id' => { 'type' => 'integer' },
            'title' => { 'type' => 'string' },
            'text' => { 'type' => 'string' },
            'created_on' => { 'type' => 'string', 'format' => 'date-time' },
            'updated_on' => { 'type' => 'string', 'format' => 'date-time' },
            'author' => { '$ref' => 'IDName' }
          }
        },
        'list' => {
          'wrapper' => 'wiki_pages',
          'envelope' => false,
          'properties' => { '__items__' => true }
        },
        'path_key' => 'title'
      },
      'Version' => {
        'request' => nil,
        'single' => {
          'wrapper' => 'version',
          'required' => %w[id name],
          'properties' => {
            'id' => { 'type' => 'integer' },
            'name' => { 'type' => 'string' },
            'status' => { 'type' => 'string' },
            'sharing' => { 'type' => 'string' },
            'due_date' => { 'type' => 'string', 'format' => 'date' },
            'description' => { 'type' => 'string' },
            'created_on' => { 'type' => 'string', 'format' => 'date-time' },
            'updated_on' => { 'type' => 'string', 'format' => 'date-time' },
            'project' => { '$ref' => 'IDName' }
          }
        },
        'list' => {
          'wrapper' => 'versions',
          'envelope' => false,
          'properties' => { '__items__' => true }
        },
        'path_key' => 'id'
      },
      'Tracker' => {
        'request' => nil,
        'single' => {
          'wrapper' => 'tracker',
          'required' => %w[id name],
          'properties' => {
            'id' => { 'type' => 'integer' },
            'name' => { 'type' => 'string' }
          }
        },
        'list' => {
          'wrapper' => 'trackers',
          'envelope' => false,
          'properties' => { '__items__' => true }
        },
        'path_key' => 'id'
      },
      'IssueStatus' => {
        'request' => nil,
        'single' => {
          'wrapper' => 'issue_status',
          'required' => %w[id name],
          'properties' => {
            'id' => { 'type' => 'integer' },
            'name' => { 'type' => 'string' },
            'is_closed' => { 'type' => 'boolean' }
          }
        },
        'list' => {
          'wrapper' => 'issue_statuses',
          'envelope' => false,
          'properties' => { '__items__' => true }
        },
        'path_key' => 'id'
      },
      'IssuePriority' => {
        'request' => nil,
        'single' => {
          'wrapper' => 'issue_priority',
          'required' => %w[id name],
          'properties' => {
            'id' => { 'type' => 'integer' },
            'name' => { 'type' => 'string' },
            'is_default' => { 'type' => 'boolean' },
            'active' => { 'type' => 'boolean' }
          }
        },
        'list' => {
          'wrapper' => 'issue_priorities',
          'envelope' => false,
          'properties' => { '__items__' => true }
        },
        'path_key' => 'id'
      },
      'IssueCategory' => {
        'request' => nil,
        'single' => {
          'wrapper' => 'issue_category',
          'required' => %w[id name],
          'properties' => {
            'id' => { 'type' => 'integer' },
            'name' => { 'type' => 'string' },
            'project' => { '$ref' => 'IDName' },
            'assigned_to' => { '$ref' => 'IDName' }
          }
        },
        'list' => {
          'wrapper' => 'issue_categories',
          'envelope' => false,
          'properties' => { '__items__' => true }
        },
        'path_key' => 'id'
      },
      'Relation' => {
        'request' => {
          'wrapper' => 'relation',
          'required' => %w[relation_type issue_to_id],
          'properties' => {
            'relation_type' => { 'type' => 'string' },
            'issue_id' => { 'type' => 'integer' },
            'issue_to_id' => { 'type' => 'integer' },
            'delay' => { 'type' => 'integer' }
          }
        },
        'single' => {
          'wrapper' => 'relation',
          'required' => %w[id issue_id issue_to_id relation_type],
          'properties' => {
            'id' => { 'type' => 'integer' },
            'issue_id' => { 'type' => 'integer' },
            'issue_to_id' => { 'type' => 'integer' },
            'relation_type' => { 'type' => 'string' },
            'delay' => { 'type' => 'integer' }
          }
        },
        'list' => {
          'wrapper' => 'relations',
          'envelope' => false,
          'properties' => { '__items__' => true }
        },
        'path_key' => 'id'
      }
    }.freeze

    # Full component schema map emitted into the spec. Produces, per resource:
    #   {Name}        — flat response object (from 'single')
    #   {Name}Single  — single-response wrapper, e.g. { issue: Issue }
    #   {Name}List    — list-response wrapper, e.g. { issues: [Issue], total_count, ... }
    #   {Name}Request — flat request object (when 'request' present)
    #   {Name}Create  — create-body wrapper, e.g. { issue: IssueRequest } (when 'request' present)
    #   {Name}Update  — update-body wrapper (when 'update' present)
    #
    # Resources without an 'update' key (everything but Issue) reuse {Name}Request
    # for their update body, matching Redmine's tolerance for extra fields.
    def self.components
      result = SHARED.dup
      RESOURCES.each do |name, res|
        single = res['single']
        result[name] = flat_schema(single)
        result["#{name}Single"] = wrapper_schema(single['wrapper'], name)

        list = res['list']
        result["#{name}List"] = list_schema(list, name)

        req = res['request']
        if req
          result["#{name}Request"] = flat_schema(req)
          result["#{name}Create"] = wrapper_schema(req['wrapper'], "#{name}Request")
        end

        upd = res['update']
        next if upd.nil?

        result["#{name}Update"] = flat_schema(upd)
        result["#{name}UpdateRequest"] = wrapper_schema(upd['wrapper'], "#{name}Update")
      end
      result
    end

    # The bare wrapper schema: { wrapper: { $ref: payload } }.
    def self.wrapper_schema(wrapper, payload_ref)
      {
        'type' => 'object',
        'required' => [wrapper],
        'properties' => {
          wrapper => { '$ref' => payload_ref }
        }
      }
    end

    def self.flat_schema(spec)
      schema = { 'type' => 'object' }
      schema['required'] = spec['required'] unless spec['required'].nil? || spec['required'].empty?
      schema['properties'] = spec['properties'].dup
      schema
    end

    def self.list_schema(spec, item_name)
      props = spec['properties'].dup
      props.delete('__items__')
      props[spec['wrapper']] = {
        'type' => 'array',
        'items' => { '$ref' => item_name }
      }
      if spec['envelope']
        props['total_count'] = { 'type' => 'integer' }
        props['offset'] = { 'type' => 'integer' }
        props['limit'] = { 'type' => 'integer' }
      end
      {
        'type' => 'object',
        'required' => [spec['wrapper']],
        'properties' => props
      }
    end
  end
end
