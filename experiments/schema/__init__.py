"""Validate numbers.json against numbers.schema.json, using only the
standard library.

WHY NOT `pip install jsonschema`. numbers.py, analyze.py and run.py are
stdlib-only by design — the statistics must be runnable anywhere a
Python interpreter exists, with nothing to install. Reproducing the six
numbers is the project's central claim, and putting a dependency
install in front of it would make the claim harder to check for exactly
the people most likely to want to check it. The experiments container
would carry it too.

WHY THIS IS NOT A TOY. A hand-written validator that quietly accepts
what it does not understand is worse than no validator, because it
reports success. Two things prevent that:

  1. UNKNOWN KEYWORDS ARE AN ERROR. This implements a subset of JSON
     Schema 2020-12, and any keyword outside that subset raises rather
     than being skipped. A schema that grows a `patternProperties` or
     an `allOf` fails loudly here instead of being half-checked.

  2. IT IS CROSS-CHECKED AGAINST A REAL IMPLEMENTATION. The fixture
     corpus in testdata/ is run through this validator by
     `python3 -m schema --selftest`, and through
     santhosh-tekuri/jsonschema (Go) by numbers_schema_test.go in
     services/ingestion. Both must reach the same verdict on every
     fixture. If this validator is wrong about a case that matters,
     that test is what says so.

Run: make numbers-schema-selftest
"""
import json
import pathlib
import re
import sys

SCHEMA_VERSION = "1.0.0"

HERE = pathlib.Path(__file__).resolve().parent
SCHEMA_PATH = HERE / "numbers.schema.json"

# Everything this validator implements. A schema using anything else is
# a schema this cannot be trusted to check.
SUPPORTED = {
    "$schema", "$id", "$defs", "$ref",
    "title", "description",
    "type", "enum", "const",
    "properties", "required", "additionalProperties",
    "items", "minItems", "maxItems",
    "minimum", "maximum",
    "minLength", "maxLength",
    "pattern",
    "oneOf",
}

TYPES = {
    "object": dict,
    "array": list,
    "string": str,
    "number": (int, float),
    "integer": int,
    "boolean": bool,
    "null": type(None),
}


class SchemaError(Exception):
    """The schema itself is unusable — not a data problem."""


def _matches_type(value, name):
    expected = TYPES.get(name)
    if expected is None:
        raise SchemaError(f"unknown type {name!r}")
    # In JSON a boolean is not a number, though in Python it is an int.
    if isinstance(value, bool) and name in ("number", "integer"):
        return False
    if name == "integer" and isinstance(value, float):
        return value.is_integer()
    return isinstance(value, expected)


def _resolve(ref, root):
    if not ref.startswith("#/"):
        raise SchemaError(f"only local refs are supported, got {ref!r}")
    node = root
    for part in ref[2:].split("/"):
        part = part.replace("~1", "/").replace("~0", "~")
        if not isinstance(node, dict) or part not in node:
            raise SchemaError(f"cannot resolve {ref!r}")
        node = node[part]
    return node


def _validate(value, schema, root, path, errors):
    if not isinstance(schema, dict):
        raise SchemaError(f"schema at {path} is not an object")

    unknown = set(schema) - SUPPORTED
    if unknown:
        raise SchemaError(
            f"schema at {path or '<root>'} uses unsupported keyword(s) "
            f"{sorted(unknown)}. This validator checks a subset deliberately; "
            f"extend SUPPORTED and implement them, or the document is only "
            f"half-checked."
        )

    where = path or "<root>"

    if "$ref" in schema:
        _validate(value, _resolve(schema["$ref"], root), root, path, errors)
        return

    if "oneOf" in schema:
        matched = 0
        # Keep the branch failures. "matches 0 of the allowed shapes" is
        # true and useless — it was the entire message when the schema
        # first met a real run, and it named neither the branch nor the
        # field. The overwhelmingly common shape here is
        # `oneOf: [null, <something>]`, so reporting why the non-null
        # branch failed is what the reader actually needs.
        branch_errors = []
        for option in schema["oneOf"]:
            sub = []
            _validate(value, option, root, path, sub)
            if not sub:
                matched += 1
            elif option.get("type") != "null":
                branch_errors.extend(sub)
        if matched != 1:
            if matched == 0 and branch_errors:
                errors.extend(branch_errors)
            else:
                errors.append(
                    f"{where}: matches {matched} of the allowed shapes, expected exactly 1")
            return

    if "type" in schema:
        names = schema["type"]
        if isinstance(names, str):
            names = [names]
        if not any(_matches_type(value, n) for n in names):
            errors.append(f"{where}: expected {' or '.join(names)}, got {type(value).__name__}")
            return

    if "const" in schema and value != schema["const"]:
        errors.append(f"{where}: must be {schema['const']!r}")

    if "enum" in schema and value not in schema["enum"]:
        errors.append(f"{where}: {value!r} is not one of {schema['enum']}")

    if isinstance(value, str):
        if "pattern" in schema and not re.search(schema["pattern"], value):
            errors.append(f"{where}: {value!r} does not match {schema['pattern']}")
        if "minLength" in schema and len(value) < schema["minLength"]:
            errors.append(
                f"{where}: is {len(value)} characters, minimum {schema['minLength']}")
        if "maxLength" in schema and len(value) > schema["maxLength"]:
            errors.append(
                f"{where}: is {len(value)} characters, maximum {schema['maxLength']}")

    if isinstance(value, (int, float)) and not isinstance(value, bool):
        if "minimum" in schema and value < schema["minimum"]:
            errors.append(f"{where}: {value} is below the minimum {schema['minimum']}")
        if "maximum" in schema and value > schema["maximum"]:
            errors.append(f"{where}: {value} is above the maximum {schema['maximum']}")

    if isinstance(value, list):
        if "minItems" in schema and len(value) < schema["minItems"]:
            errors.append(f"{where}: has {len(value)} items, minimum {schema['minItems']}")
        if "maxItems" in schema and len(value) > schema["maxItems"]:
            errors.append(f"{where}: has {len(value)} items, maximum {schema['maxItems']}")
        if "items" in schema:
            for i, item in enumerate(value):
                _validate(item, schema["items"], root, f"{path}[{i}]", errors)

    if isinstance(value, dict):
        for name in schema.get("required", []):
            if name not in value:
                errors.append(f"{where}: missing required property {name!r}")

        properties = schema.get("properties", {})
        for name, item in value.items():
            child = f"{path}.{name}" if path else name
            if name in properties:
                _validate(item, properties[name], root, child, errors)
                continue
            extra = schema.get("additionalProperties", True)
            if extra is False:
                errors.append(f"{where}: unexpected property {name!r}")
            elif isinstance(extra, dict):
                _validate(item, extra, root, child, errors)


def load_schema():
    try:
        return json.loads(SCHEMA_PATH.read_text())
    except OSError as e:
        raise SchemaError(f"cannot read {SCHEMA_PATH}: {e}") from e


def validate(document, schema=None):
    """Return a list of human-readable errors; empty means valid."""
    schema = schema if schema is not None else load_schema()
    errors = []
    _validate(document, schema, schema, "", errors)
    return errors


def validate_numbers(numbers):
    """Validate a numbers.json document. Used by numbers.py before it
    writes, so a malformed measurement is never published."""
    return validate(numbers)


# ---------------------------------------------------------------------
# selftest
# ---------------------------------------------------------------------

TESTDATA = HERE / "testdata"


def _fixtures():
    """Yield (path, should_be_valid) for every committed fixture."""
    for expectation, valid in (("valid", True), ("invalid", False)):
        directory = TESTDATA / expectation
        for path in sorted(directory.glob("*.json")):
            yield path, valid


def selftest():
    failures = 0
    total = 0
    for path, should_be_valid in _fixtures():
        total += 1
        document = json.loads(path.read_text())
        errors = validate(document)
        ok = (not errors) if should_be_valid else bool(errors)
        if not ok:
            failures += 1
            if should_be_valid:
                print(f"  FAIL  {path.name} should be VALID but got:")
                for e in errors:
                    print(f"          {e}")
            else:
                print(f"  FAIL  {path.name} should be INVALID but passed")
        else:
            detail = "" if should_be_valid else f"  ({errors[0]})"
            print(f"  ok    {path.name}{detail}")

    if total == 0:
        print("  no fixtures found — the selftest is not testing anything")
        return 1
    print(f"\n  {total - failures}/{total} schema fixtures behave as expected")
    return 1 if failures else 0


if __name__ == "__main__":
    if "--selftest" in sys.argv:
        sys.exit(selftest())
    print(__doc__)
    sys.exit(2)
