#!/usr/bin/env python3
"""Export synthetic lessons or measure a local model. Never executes model output."""
import argparse
import hashlib
import json
import math
from pathlib import Path
import re
import time
import unicodedata
import urllib.request

from curriculum import ACTIONS, CLARIFIED_SYSTEM, EVAL, LABELS, SYSTEM, TRAIN, VERSION, messages, validate

PROFILES = {"baseline": SYSTEM, "defined-labels-v1": CLARIFIED_SYSTEM,
            "lessons-v1": CLARIFIED_SYSTEM}


def suite_data(name):
    if name == "review-v1":
        validate()
        return TRAIN, EVAL
    if name == "extended-v2":
        import extended_curriculum
        extended_curriculum.validate()
        return extended_curriculum.TRAIN, extended_curriculum.EVAL
    raise ValueError("Unknown suite: " + name)


def terms(text):
    text = "".join(c for c in unicodedata.normalize("NFD", text.lower())
                   if unicodedata.category(c) != "Mn")
    return {x for x in re.findall(r"\w+", text) if len(x) > 3}


def select_lessons(item, training, count=3, char_budget=6000):
    """Rank only question/evidence overlap; never consult the evaluated answer."""
    from curriculum import prompt
    query = terms(prompt(item))
    indexed = [(x, terms(prompt(x))) for x in training]
    frequencies = {term: sum(term in ts for _, ts in indexed) for term in query}
    def score(entry):
        return sum(math.log((len(training) + 1) / (frequencies[t] + 1)) + 1
                   for t in query & entry[1])
    selected = []
    for row, tokens in sorted(indexed, key=lambda x: (-score(x), x[0]["id"])):
        if not query & tokens:
            continue
        size = len(prompt(row)) + len(json.dumps(row["answer"], ensure_ascii=False))
        if size > char_budget:
            continue
        selected.append(row)
        char_budget -= size
        if len(selected) == count:
            break
    return selected


def teaching_messages(item, training, system, profile):
    from curriculum import prompt
    selected = select_lessons(item, training) if profile == "lessons-v1" else []
    result = [{"role": "system", "content": system}]
    for example in selected:
        result.extend([{"role": "user", "content": prompt(example)},
                       {"role": "assistant", "content": json.dumps(example["answer"], ensure_ascii=False)}])
    result.append({"role": "user", "content": prompt(item)})
    return result, [x["id"] for x in selected]

SCHEMA = {"type": "object", "additionalProperties": False,
          "required": ["finding", "evidence_ids", "action", "explanation"],
          "properties": {"finding": {"type": "string", "enum": LABELS},
                         "action": {"type": "string", "enum": ACTIONS},
                         "evidence_ids": {"type": "array", "items": {"type": "string"}},
                         "explanation": {"type": "string"}}}


def write_json(path, data):
    # Refuse to replace previous evidence.
    with path.open("x", encoding="utf-8") as file:
        json.dump(data, file, ensure_ascii=False, indent=2)


def fingerprint(data):
    return hashlib.sha256(json.dumps(data, sort_keys=True, ensure_ascii=False).encode()).hexdigest()


def export(out, suite="review-v1"):
    training, _ = suite_data(suite)
    export_system = CLARIFIED_SYSTEM if suite == "extended-v2" else SYSTEM
    with (out / "train.jsonl").open("x", encoding="utf-8") as file:
        for item in training:
            row = {"messages": messages(item, export_system) + [{"role": "assistant", "content": json.dumps(item["answer"], ensure_ascii=False)}]}
            file.write(json.dumps(row, ensure_ascii=False) + "\n")
    write_json(out / "provenance.json", {"version": suite, "train_sha256": fingerprint(training),
               "examples": [{k: v for k, v in x.items() if k in {"id", "family", "teacher", "editor", "source", "review_status", "references"}} for x in training],
               "weights_trained": False, "note": "Seed dataset, not sufficient for specialization."})


def grade(answer, expected, evidence):
    fields = set(SCHEMA["required"])
    valid = (isinstance(answer, dict) and set(answer) == fields
             and isinstance(answer.get("finding"), str) and answer["finding"] in LABELS
             and isinstance(answer.get("action"), str) and answer["action"] in ACTIONS
             and isinstance(answer.get("explanation"), str) and bool(answer["explanation"].strip())
             and isinstance(answer.get("evidence_ids"), list)
             and all(isinstance(x, str) for x in answer["evidence_ids"]))
    if not valid:
        return {"schema": False, "finding": False, "action": False, "evidence": False, "pass": False}
    ids = answer["evidence_ids"]
    result = {"schema": True, "finding": answer["finding"] == expected["finding"],
              "action": answer["action"] == expected["action"],
              "evidence": len(ids) == len(set(ids)) and set(ids) == set(expected["evidence_ids"])
              and set(ids) <= {x["id"] for x in evidence}}
    result["pass"] = all(result.values())
    return result


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        raise ValueError("Local Ollama redirects are not allowed")


def request(path, payload=None, timeout=120):
    # Fixed loopback endpoint; do not send exercises via proxy environment variables.
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    data = None if payload is None else json.dumps(payload).encode()
    req = urllib.request.Request("http://127.0.0.1:11434" + path, data=data,
                                 headers={"Content-Type": "application/json"})
    with opener.open(req, timeout=timeout) as response:
        return json.load(response)


def evaluate(out, model, limit, timeout, profile="baseline", suite="review-v1"):
    system = PROFILES[profile]
    training, evaluation = suite_data(suite)
    if not 1 <= limit <= len(evaluation):
        raise ValueError("Invalid evaluation limit")
    available = request("/api/tags")
    info = next((x for x in available["models"] if x["name"] == model), None)
    if info is None:
        raise ValueError("Model must already be installed locally: " + model)
    configuration = request("/api/show", {"model": model})
    if configuration.get("system"):
        system = configuration["system"] + "\n\nFor this review exercise, use the following JSON output contract instead of the general guidance format:\n" + system
    metadata = {"version": suite, "model": info, "ollama": request("/api/version"),
                "model_configuration": configuration,
                "evaluation_sha256": fingerprint(evaluation), "system_sha256": fingerprint(system),
                "train_sha256": fingerprint(training),
                "runner_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
                "profile": profile,
                "started_at": time.time(), "weights_trained": False,
                "options": {"temperature": 0, "seed": 42, "num_ctx": 4096, "num_predict": 512},
                "think": False, "limit": limit}
    write_json(out / "metadata.json", metadata)
    results = []
    for item in evaluation[:limit]:
        start = time.monotonic()
        model_messages, lesson_ids = teaching_messages(item, training, system, profile)
        record = {"id": item["id"], "family": item["family"], "lesson_ids": lesson_ids,
                  "request_sha256": fingerprint(model_messages), "expected_finding": item["answer"]["finding"]}
        try:
            raw = request("/api/chat", {"model": model, "messages": model_messages, "format": SCHEMA,
                                      "stream": False, "think": False, "options": metadata["options"]}, timeout)
            record["raw"] = raw
            if raw.get("done") is not True or raw.get("done_reason") == "length":
                raise ValueError("Incomplete generation")
            answer = json.loads(raw["message"]["content"])
            record["answer"] = answer
            record["grade"] = grade(answer, item["answer"], item["evidence"])
        except (OSError, ValueError, KeyError, TypeError) as exc:
            record["error"] = type(exc).__name__ + ": " + str(exc)
            record["grade"] = grade(None, item["answer"], item["evidence"])
        record["elapsed_seconds"] = round(time.monotonic() - start, 3)
        write_json(out / (item["id"] + ".json"), record)
        results.append(record)
        print(json.dumps({k: record[k] for k in ("id", "grade", "elapsed_seconds")}), flush=True)
    summary = {"total": len(results), "passed": sum(x["grade"]["pass"] for x in results),
               "errors": sum("error" in x for x in results), "weights_trained": False,
               "profile": profile, "suite": suite,
               "classification_correct": sum(x["grade"]["finding"] for x in results),
               "false_positives": sum(x["expected_finding"] == "no_finding" and
                    isinstance(x.get("answer"), dict) and x["answer"].get("finding") in LABELS[:-2] for x in results),
               "unsupported_claims": sum(x["expected_finding"] == "insufficient_evidence" and
                    isinstance(x.get("answer"), dict) and x["answer"].get("finding") in LABELS[:-2] for x in results),
               "prompt_tokens": sum(x.get("raw", {}).get("prompt_eval_count", 0) for x in results),
               "output_tokens": sum(x.get("raw", {}).get("eval_count", 0) for x in results),
               "elapsed_seconds": round(sum(x["elapsed_seconds"] for x in results), 3),
               "scope": "synthetic classification, remediation category and evidence selection only",
               "explanations_semantically_graded": False, "nexus_agent_evaluated": False}
    write_json(out / "summary.json", summary)
    print(json.dumps(summary), flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=["export", "evaluate"])
    parser.add_argument("--out", type=Path, required=True, help="New output directory")
    parser.add_argument("--model", default="qwen3:4b")
    parser.add_argument("--limit", type=int)
    parser.add_argument("--timeout", type=int, default=120)
    parser.add_argument("--profile", choices=PROFILES, default="baseline")
    parser.add_argument("--suite", choices=["review-v1", "extended-v2"], default="review-v1")
    args = parser.parse_args()
    _, evaluation = suite_data(args.suite)
    if args.limit is None:
        args.limit = len(evaluation)
    if not 1 <= args.limit <= len(evaluation) or not 1 <= args.timeout <= 600:
        parser.error("Invalid limit or timeout")
    validate()
    args.out.mkdir(parents=True, mode=0o700, exist_ok=False)
    if args.command == "export":
        export(args.out, args.suite)
    else:
        evaluate(args.out, args.model, args.limit, args.timeout, args.profile, args.suite)


if __name__ == "__main__":
    main()
