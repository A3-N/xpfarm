# xpfarm
penits/penaar

```python
# app/modules/demo_min.py
from __future__ import annotations
import datetime, uuid
from sqlalchemy import text

MODULE  = "Demo Minimal"
KEY     = "demo_min"
VERSION = "1.0.0"

def run():
    # 1) DB session (uses project’s SQLAlchemy)
    from app.db import SessionLocal, engine  # EDIT IF NEEDED: path to SessionLocal/engine
    session = SessionLocal()

    # 2) Ensure results table for this module exists
    table = f"results_{KEY}"
    ddl = f"""
    CREATE TABLE IF NOT EXISTS {table} (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      batch_id   TEXT NOT NULL,
      created_at TEXT NOT NULL,
      module     TEXT NOT NULL,
      -- your data columns:
      target     TEXT,
      note       TEXT
    );
    """
    session.execute(text(ddl))

    # 3) Read scope (supports either 'kind' or 'type')
    cols = [r[1] for r in session.execute(text("PRAGMA table_info('scope_items')"))]
    kind_col = "kind" if "kind" in cols else ("type" if "type" in cols else None)
    sel = "id, value" + (f", {kind_col}" if kind_col else "")
    scope = session.execute(text(f"SELECT {sel} FROM scope_items")).mappings().all()

    # 4) Prepare rows for this run
    batch_id = datetime.datetime.utcnow().strftime("%Y%m%d-%H%M%S") + "-" + str(uuid.uuid4())[:8]
    now = datetime.datetime.utcnow().isoformat(timespec="seconds") + "Z"

    rows = []
    for s in scope:
        k = (s.get(kind_col) or "").lower() if kind_col else ""
        v = (s.get("value") or "").strip()
        if not v:
            continue
        # EDIT IF NEEDED: your selection logic (example: only domains/hosts)
        if k in ("domain", "host", "ip", "cidr", "url") or not k:
            rows.append({
                "batch_id": batch_id,
                "created_at": now,
                "module": KEY,
                "target": v,
                "note": f"seen_in_scope:{k or 'unknown'}"
            })

    # 5) Bulk insert (only if we have rows)
    if rows:
        cols = ("batch_id, created_at, module, target, note")
        session.execute(
            text(f"INSERT INTO {table} ({cols}) VALUES (:batch_id,:created_at,:module,:target,:note)"),
            rows
        )

    session.commit()
    session.close()

    # 6) Minimal summary (module_loader prints/records this)
    return {
        "module": MODULE,
        "key": KEY,
        "version": VERSION,
        "batch_id": batch_id,
        "rows_written": len(rows),
    }
```
