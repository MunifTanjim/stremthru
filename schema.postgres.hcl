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

table "magnet_cache_file" {
  schema = schema.public

  column "h" {
    null = false
    type = varchar
  }
  column "n" {
    null = false
    type = varchar
  }
  column "i" {
    null = false
    type = int
    default = -1
  }
  column "s" {
    null = false
    type = bigint
    default = -1
  }
  column "sid" {
    null = false
    type = varchar
    default = ""
  }
  primary_key {
    columns = [column.h, column.n]
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

table "kv" {
  schema = schema.public

  column "t" {
    null = false
    type = text
    default = ""
  }
  column "k" {
    null = false
    type = text
  }
  column "v" {
    null = false
    type = text
  }
  column "cat" {
    null = false
    type = timestamptz
    default = sql("current_timestamp")
  }
  column "uat" {
    null = false
    type = timestamptz
    default = sql("current_timestamp")
  }
  primary_key {
    columns = [column.t, column.k]
  }
}
