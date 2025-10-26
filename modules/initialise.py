# modules/initialise.py
"""
Initialise module with robust rules:

- Only processes UNVERIFIED scope items.
- Classifies into: "IP Address" | "CIDR" | "Domain" | "Wildcard" | "Other Asset" | "AI Model" | "API"
- Normalizes values (domains lowercased, trailing dot stripped; /32,/128 normalized to IP).
- Special-case for API:
    * Accepts full URLs or bare hosts.
    * If hostname is a domain and resolves => KEEP in API (don't move).
    * If hostname is an IP => KEEP in API.
    * If value/host is a CIDR => MOVE to CIDR.
    * If domain fails to resolve => REMOVE.
- Elsewhere:
    * Wrong bucket -> MOVE to canonical type.
    * Domain must resolve; if not => REMOVE.
- Other Asset:
    * If value parses as any canonical type => MOVE to that type; else KEEP.
- AI Model: KEEP (no parsing).
- Logs actions and marks verified.

Idempotent via verified flag.
"""

from datetime import datetime
import ipaddress
import re
import socket
from urllib.parse import urlparse
from typing import Optional, Tuple

from app.db import SessionLocal
from app.models import ScopeItem
from app.metrics import record_metric

NAME = "initialise"
DESCRIPTION = "Normalize & verify scope entries with API/domain exceptions; move/remove accordingly."

# --------- Patterns ---------
# Accept one or more labels, last label 2-63 chars; optional trailing dot
RE_FQDN = re.compile(r"^(?=.{1,253}\.?$)(?:[A-Za-z0-9-]{1,63}\.)+[A-Za-z0-9-]{2,63}\.?$")
RE_WILDCARD = re.compile(r"^\*\.(?=.{1,253}\.?$)(?:[A-Za-z0-9-]{1,63}\.)+[A-Za-z0-9-]{2,63}\.?$")

CANON_TYPES = {"CIDR", "Domain", "Wildcard", "IP Address", "Other Asset", "URL", "API"}

def _strip_trailing_dot(s: str) -> str:
    return s.rstrip(".")

def _is_ip(val: str) -> Optional[ipaddress._BaseAddress]:
    try:
        return ipaddress.ip_address(val.strip())
    except Exception:
        return None

def _as_cidr_if_any(val: str) -> Optional[ipaddress._BaseNetwork]:
    try:
        return ipaddress.ip_network(val.strip(), strict=False)
    except Exception:
        return None

def _is_fqdn(val: str) -> bool:
    s = _strip_trailing_dot(val.strip().lower())
    if any(x in s for x in (" ", "/", ":")):
        return False
    return bool(RE_FQDN.match(s))

def _is_wildcard(val: str) -> bool:
    s = val.strip().lower()
    if any(x in s for x in (" ", "/", ":")):
        return False
    return bool(RE_WILDCARD.match(s))

def _dns_resolves(host: str) -> bool:
    name = _strip_trailing_dot(host.strip())
    try:
        socket.getaddrinfo(name, None)
        return True
    except Exception:
        return False

def _normalize_domain(val: str) -> str:
    return _strip_trailing_dot(val.strip().lower())

def _classify_value(raw: str) -> Tuple[Optional[str], str]:
    """
    Return (canonical_type, normalized_value) or (None, raw) if invalid.
    """
    v = raw.strip()
    if not v:
        return None, raw

    # Wildcard first
    if _is_wildcard(v):
        return "Wildcard", _normalize_domain(v)

    # Pure IP?
    ipobj = _is_ip(v)
    if ipobj:
        return "IP Address", str(ipobj)

    # CIDR?
    net = _as_cidr_if_any(v)
    if net:
        # Normalize textual form
        txt = str(net)
        # If it's effectively a single host (/32 or /128), treat as IP (per request)
        if (net.version == 4 and net.prefixlen == 32) or (net.version == 6 and net.prefixlen == 128):
            return "IP Address", str(net.network_address)
        return "CIDR", txt

    # FQDN?
    if _is_fqdn(v):
        return "Domain", _normalize_domain(v)

    # Could be URL: we don't classify as URL; classification uses the hostname later in API logic
    return None, v

def _api_extract_host(val: str) -> str:
    """
    For API entries, accept URLs or bare hosts.
    If URL, return hostname (lowercased, no trailing dot). Else return value as-is (normalized).
    """
    v = val.strip()
    parsed = urlparse(v if "://" in v else f"http://{v}")
    host = parsed.hostname or v
    return _normalize_domain(host)

# --- add/replace helpers ---

def _canonicalize(value_raw: str, stype_hint: str | None = None) -> tuple[Optional[str], Optional[str], Optional[str]]:
    """
    Return (canonical_type, canonical_value, api_host_key)
    - canonical_type in {"IP Address","CIDR","Domain","Wildcard","URL","API","Other Asset"} or None if invalid
    - canonical_value: normalized stored value for that type
    - api_host_key: for API rows only, a normalized host key used for dedupe; else None
    """
    v = value_raw.strip()
    if not v:
        return None, None, None

    # API: extract host for dedupe key, but keep the original value if we keep the row
    if stype_hint == "API":
        host = _api_extract_host(v)
        ctype, cval = _classify_value(host)
        if ctype == "CIDR":
            # will be moved to CIDR, no API dedupe key needed
            return "CIDR", cval, None
        if ctype == "IP Address":
            # keep in API; canonical value is *original* v (may be URL), host key dedupes
            return "API", v, f"ip:{cval}"
        if ctype == "Domain":
            # must resolve; we'll enforce in verification
            return "API", v, f"host:{cval}"
        if ctype == "Wildcard":
            return "API", v, f"wild:{cval}"
        # Not a recognized host → still API free-form (dedupe by lowered original)
        return "API", v, f"str:{v.lower()}"

    # Non-API classification/normalization
    ctype, cval = _classify_value(v)
    if not ctype:
        # Maybe it's a URL and the user mislabeled; treat as URL canonical
        if stype_hint == "URL":
            return "URL", v, None
        return None, None, None
    if ctype in ("Domain", "Wildcard", "IP Address", "CIDR"):
        return ctype, cval, None
    if stype_hint == "URL":
        return "URL", v, None
    if stype_hint == "Other Asset":
        # keep as Other Asset if not canonical
        return "Other Asset", v, None
    return ctype, cval, None


def _canonical_key_for_item(it: ScopeItem) -> tuple[int, str, str]:
    """
    Build a canonical dedupe key for an existing row.
    - For API rows, dedupe by host key (derived) but still store original value.
    - For others, dedupe by normalized stored value.
    """
    stype = (it.stype or "").strip()
    raw = (it.value or "").strip()
    if stype == "API":
        # calculate host key
        host = _api_extract_host(raw)
        ctype, cval = _classify_value(host)
        if ctype == "CIDR":    # Shouldn't live under API; will be moved later
            return (it.target_id, "CIDR", cval.lower())
        if ctype == "IP Address":
            return (it.target_id, "API", f"ip:{cval}")
        if ctype == "Domain":
            return (it.target_id, "API", f"host:{cval}")
        if ctype == "Wildcard":
            return (it.target_id, "API", f"wild:{cval}")
        return (it.target_id, "API", f"str:{raw.lower()}")
    else:
        # normalize by the same rules as verification
        ctype, cval = _classify_value(raw)
        if not ctype:
            # non-canonical: for URL/Other Asset, dedupe by lowered raw
            if stype in ("URL", "Other Asset"):
                return (it.target_id, stype, raw.lower())
            # invalid; verification step will delete
            return (it.target_id, stype, raw.lower())
        return (it.target_id, ctype, (cval or raw).lower())

def run():
    db = SessionLocal()
    moved = removed = verified = examined = 0
    notes = []
    try:
        # Load ALL rows for dedupe (verified + unverified)
        all_rows = db.query(ScopeItem).order_by(ScopeItem.id.asc()).all()

        # Global canonical dedupe map: (target_id, ctype, cvalue_key) -> kept_id
        canon_seen: dict[tuple[int,str,str], int] = {}
        dupes_removed = 0

        for r in all_rows:
            key = _canonical_key_for_item(r)
            kept_id = canon_seen.get(key)
            if kept_id is None:
                canon_seen[key] = r.id
                continue
            # Duplicate → delete newer one (this r, since we iterate id asc)
            notes.append(f"REMOVE [{r.id}] duplicate of [{kept_id}] ({r.stype} '{r.value}')")
            db.delete(r)
            dupes_removed += 1

        if dupes_removed:
            db.commit()
            record_metric(db, NAME, "duplicates_removed", value_num=float(dupes_removed))
            notes.append(f"Removed {dupes_removed} duplicates globally")

        # Proceed with UNVERIFIED processing
        q = db.query(ScopeItem).filter((ScopeItem.verified == False) | (ScopeItem.verified.is_(None)))
        items = q.order_by(ScopeItem.created_at.asc(), ScopeItem.id.asc()).all()

        for it in items:
            examined += 1
            original_type = it.stype
            value_raw = (it.value or "").strip()

            # API special-case first
            if original_type == "API":
                host = _api_extract_host(value_raw)
                canonical, norm = _classify_value(host)
                if canonical == "CIDR":
                    notes.append(f"MOVE  [{it.id}] '{value_raw}': API → CIDR")
                    it.stype = "CIDR"
                    it.value = norm
                    it.verified = True
                    it.verified_at = datetime.utcnow()
                    it.verified_note = "moved from API to CIDR"
                    db.add(it)
                    moved += 1; verified += 1
                    continue
                if canonical == "IP Address":
                    it.verified = True; it.verified_at = datetime.utcnow()
                    it.verified_note = "API host=ip ok"
                    db.add(it); verified += 1
                    continue
                if canonical == "Domain":
                    if not _dns_resolves(norm):
                        notes.append(f"REMOVE [{it.id}] API domain '{norm}' (DNS NX)")
                        db.delete(it); removed += 1
                        continue
                    it.verified = True; it.verified_at = datetime.utcnow()
                    it.verified_note = "API host=domain ok (DNS)"
                    db.add(it); verified += 1
                    continue
                if canonical == "Wildcard":
                    it.verified = True; it.verified_at = datetime.utcnow()
                    it.verified_note = "API host=wildcard ok"
                    db.add(it); verified += 1
                    continue
                # free-form API string
                it.verified = True; it.verified_at = datetime.utcnow()
                it.verified_note = "API string ok"
                db.add(it); verified += 1
                continue

            # Non-API path
            canonical, norm, _ = _canonicalize(value_raw, original_type)
            if canonical is None:
                notes.append(f"REMOVE [{it.id}] '{value_raw}' (invalid format) from {original_type}")
                db.delete(it); removed += 1
                continue

            # Domain must resolve
            if canonical == "Domain" and not _dns_resolves(norm):
                notes.append(f"REMOVE [{it.id}] '{value_raw}' (DNS NX) from {original_type}")
                db.delete(it); removed += 1
                continue

            if it.stype != canonical:
                notes.append(f"MOVE  [{it.id}] '{value_raw}': {it.stype} → {canonical}")
                it.stype = canonical
                moved += 1

            # normalize stored value
            if canonical in ("Domain", "Wildcard", "IP Address", "CIDR"):
                it.value = norm

            it.verified = True
            it.verified_at = datetime.utcnow()
            it.verified_note = (
                "Domain ok (DNS)" if canonical == "Domain" else
                "Wildcard ok" if canonical == "Wildcard" else
                "IP ok" if canonical == "IP Address" else
                "CIDR ok" if canonical == "CIDR" else "ok"
            )
            db.add(it); verified += 1

        db.commit()

        record_metric(db, NAME, "examined", value_num=float(examined))
        record_metric(db, NAME, "moved", value_num=float(moved))
        record_metric(db, NAME, "removed", value_num=float(removed))
        record_metric(db, NAME, "verified", value_num=float(verified))

        summary = (
            f"initialise: examined={examined}, moved={moved}, removed={removed}, verified={verified}\n" +
            "\n".join(notes[-400:])
        )
        return summary or "initialise: no changes"
    finally:
        db.close()

