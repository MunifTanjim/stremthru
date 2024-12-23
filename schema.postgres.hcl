schema "public" {
  comment = "standard public schema"
}

table "magnet_cache" {
  schema = schema.public

  column "store" {
    null = false
    type = varchar
  }
  column "hash" {
    null = false
    type = varchar
  }
  column "is_cached" {
    null = false
    type = bool
    default = false
  }
  column "modified_at" {
    null = false
    type = timestamptz
    default = sql("current_timestamp")
  }
  column "files" {
    null = false
    type = json
    default = "[]"
  }
  primary_key {
    columns = [column.store, column.hash]
  }
}

table "peer_token" {
  schema = schema.public

  column "id" {
    null = false
    type = varchar
  }
  column "name" {
    null = false
    type = varchar
  }
  column "created_at" {
    null = false
    type = timestamptz
    default = sql("current_timestamp")
  }
  primary_key {
    columns = [column.id]
  }
}
