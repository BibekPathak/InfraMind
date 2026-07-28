"""Deterministic health score calculation engine.

Computes a 0-100 health score based on weighted telemetry factors.
No ML dependencies — transparent and explainable.
"""

from dataclasses import dataclass
from typing import List, Tuple


@dataclass
class HealthFactor:
    name: str
    impact: float


def calculate_health_score(
    temperature: float,
    current: float,
    voltage: float,
    humidity: float,
) -> Tuple[float, str, List[HealthFactor]]:
    score = 100.0
    factors: List[HealthFactor] = []

    temp_penalty = max(0, (temperature - 75) * 1.5)
    temp_penalty = min(temp_penalty, 40)
    factors.append(HealthFactor(name="temperature", impact=-round(temp_penalty, 1)))
    score -= temp_penalty

    current_penalty = max(0, (current - 120) * 0.3)
    current_penalty = min(current_penalty, 30)
    factors.append(HealthFactor(name="current", impact=-round(current_penalty, 1)))
    score -= current_penalty

    humidity_penalty = min(humidity * 0.2, 10)
    factors.append(HealthFactor(name="humidity", impact=-round(humidity_penalty, 1)))
    score -= humidity_penalty

    score = max(0, min(100, score))

    if score > 80:
        level = "healthy"
    elif score > 50:
        level = "warning"
    else:
        level = "critical"

    return score, level, factors
