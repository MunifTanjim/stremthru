schema "main" {
}

table "store_magnet_cache" {
  schema = schema.main

  column "store" {
    null = false
    type = varchar
  }
  column "hash" {
    null = false
    type = varchar
  }
  column "touched_at" {
    null = false
    type = datetime
    default = sql("current_timestamp")
  }
  column "files" {
    null = false
    type = jsonb
    default = sql("jsonb('[]')")
  }
  primary_key {
    columns = [column.store, column.hash]
  }
}
