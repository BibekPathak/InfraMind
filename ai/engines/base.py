"""Base engine class and shared models for all AI engines."""

from dataclasses import dataclass, field
from typing import List, Optional


@dataclass
class TelemetryInput:
    temperature: float = 0
    current: float = 0
    voltage: float = 0
    humidity: float = 0
    history: List[dict] = field(default_factory=list)


@dataclass
class HealthFactor:
    name: str
    impact: float
    details: str = ""


@dataclass
class Anomaly:
    metric: str
    severity: str  # info, warning, critical
    description: str
    value: float
    expected: float


@dataclass
class FailurePrediction:
    time_to_warning_hours: Optional[float] = None
    time_to_critical_hours: Optional[float] = None
    confidence: float = 0
    trend_direction: str = "stable"  # rising, falling, stable


@dataclass
class Recommendation:
    priority: str  # low, medium, high
    action: str
    reason: str
    estimated_cost: str = "unknown"
    # Optional executable action metadata. When set, the backend may
    # propose an autonomous action (type: command/restart/config_change/notification).
    action_type: Optional[str] = None
    action_payload: Optional[dict] = None


@dataclass
class AnalysisResult:
    health_score: float = 0
    health_level: str = "unknown"
    health_factors: List[HealthFactor] = field(default_factory=list)
    anomalies: List[Anomaly] = field(default_factory=list)
    failure_prediction: Optional[FailurePrediction] = None
    recommendations: List[Recommendation] = field(default_factory=list)


class BaseEngine:
    name: str = "base"

    def analyze(self, telemetry: TelemetryInput) -> AnalysisResult:
        raise NotImplementedError
