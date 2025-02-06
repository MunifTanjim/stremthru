schema "main" {
}

table "magnet_cache" {
  schema = schema.main

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
    type = datetime
    default = sql("unixepoch()")
  }
  column "files" {
    null = false
    type = json
    default = sql("json('[]')")
  }
  primary_key {
    columns = [column.store, column.hash]
  }
}

table "magnet_cache_file" {
  schema = schema.main

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
    type = int
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
  schema = schema.main

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
    type = datetime
    default = sql("unixepoch()")
  }
  primary_key {
    columns = [column.id]
  }
}

table "kv" {
  schema = schema.main

  column "t" {
    null = false
    type = varchar
    default = ""
  }
  column "k" {
    null = false
    type = varchar
  }
  column "v" {
    null = false
    type = varchar
  }
  column "cat" {
    null = false
    type = datetime
    default = sql("unixepoch()")
  }
  column "uat" {
    null = false
    type = datetime
    default = sql("unixepoch()")
  }
  primary_key {
    columns = [column.t, column.k]
  }
}
