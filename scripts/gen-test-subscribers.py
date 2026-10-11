#!/usr/bin/env python3
"""Generate a CSV of test subscribers for listnun import testing."""

import csv
import json
import random
import sys

COUNT = int(sys.argv[1]) if len(sys.argv) > 1 else 10000
OUT   = sys.argv[2] if len(sys.argv) > 2 else "test-subscribers.csv"

FIRST_NAMES = [
    "Alice", "Bob", "Carol", "David", "Eve", "Frank", "Grace", "Henry",
    "Iris", "Jack", "Karen", "Leo", "Maya", "Noah", "Olivia", "Peter",
    "Quinn", "Rose", "Sam", "Tara", "Uma", "Victor", "Wendy", "Xander",
    "Yara", "Zoe", "Aaron", "Beth", "Chris", "Diana",
]
LAST_NAMES = [
    "Smith", "Jones", "Brown", "Taylor", "Wilson", "Davis", "Evans",
    "Thomas", "Roberts", "Johnson", "White", "Martin", "Garcia", "Lee",
    "Walker", "Hall", "Allen", "Young", "King", "Wright", "Scott",
    "Green", "Baker", "Adams", "Nelson", "Carter", "Mitchell", "Turner",
]
DOMAINS = [
    "gmail.com", "yahoo.com", "outlook.com", "hotmail.com", "icloud.com",
    "proton.me", "fastmail.com", "zoho.com", "example.com", "test.org",
]
JOBS = ["developer", "designer", "manager", "analyst", "writer", "teacher"]
CITIES = ["New York", "London", "Tokyo", "Sydney", "Berlin", "Paris", "Lagos"]

with open(OUT, "w", newline="") as f:
    writer = csv.writer(f)
    writer.writerow(["email", "name", "attributes"])

    seen = set()
    written = 0
    attempt = 0

    while written < COUNT:
        attempt += 1
        first = random.choice(FIRST_NAMES)
        last  = random.choice(LAST_NAMES)
        domain = random.choice(DOMAINS)
        tag = attempt  # guarantee uniqueness
        email = f"{first.lower()}.{last.lower()}{tag}@{domain}"

        if email in seen:
            continue
        seen.add(email)

        name = f"{first} {last}"
        attribs = json.dumps({
            "job":  random.choice(JOBS),
            "city": random.choice(CITIES),
            "score": random.randint(1, 100),
        })

        writer.writerow([email, name, attribs])
        written += 1

print(f"wrote {written} rows to {OUT}")
