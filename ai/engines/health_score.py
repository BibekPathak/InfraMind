"""Advanced health score engine with weighted scoring and trend detection.

Asset-type aware: thresholds and weights are loaded from the asset type
registry, falling back to transformer defaults for unknown types.
"""

from engines.base import BaseEngine, TelemetryInput, AnalysisResult, HealthFactor
from engines.asset_types import get_config


class HealthScoreEngine(BaseEngine):
    name = "health_score"

    def __init__(self, asset_type: str = "transformer", trend_rate_threshold: float = 2.0):
        cfg = get_config(asset_type)
        self.asset_type = cfg.type
        self.temp_threshold = cfg.temp_warning
        self.current_threshold = cfg.current_warning
        self.voltage_min = cfg.voltage_min
        self.humidity_max = cfg.humidity_max
        self.weights = cfg.weights
        self.trend_rate_threshold = trend_rate_threshold

    def analyze(self, telemetry: TelemetryInput) -> AnalysisResult:
        t = telemetry.temperature
        c = telemetry.current
        v = telemetry.voltage
        h = telemetry.humidity
        history = telemetry.history

        result = AnalysisResult()
        score = 100.0
        factors = []

        wt = self.weights.get("temperature", 0.4)
        temp_penalty = max(0, (t - self.temp_threshold) * 1.5)
        temp_penalty = min(temp_penalty, 40)
        factors.append(HealthFactor(
            name="temperature",
            impact=-round(temp_penalty * wt, 1),
            details=f"{t}°C exceeds {self.temp_threshold}°C (weight {wt})",
        ))
        score -= temp_penalty * wt

        wc = self.weights.get("current", 0.3)
        current_penalty = max(0, (c - self.current_threshold) * 0.3)
        current_penalty = min(current_penalty, 30)
        factors.append(HealthFactor(
            name="current",
            impact=-round(current_penalty * wc, 1),
            details=f"{c}A exceeds {self.current_threshold}A (weight {wc})",
        ))
        score -= current_penalty * wc

        wv = self.weights.get("voltage", 0.15)
        voltage_penalty = 0
        if v > 0 and self.voltage_min > 0 and v < self.voltage_min:
            voltage_penalty = min((self.voltage_min - v) * 0.05, 15)
        factors.append(HealthFactor(
            name="voltage",
            impact=-round(voltage_penalty * wv, 1),
            details=f"{v}V below {self.voltage_min}V minimum (weight {wv})",
        ))
        score -= voltage_penalty * wv

        wh = self.weights.get("humidity", 0.15)
        humidity_penalty = 0
        if h > self.humidity_max:
            humidity_penalty = min((h - self.humidity_max) * 0.5, 10)
        factors.append(HealthFactor(
            name="humidity",
            impact=-round(humidity_penalty * wh, 1),
            details=f"{h}% exceeds {self.humidity_max}% threshold (weight {wh})",
        ))
        score -= humidity_penalty * wh

        trend_penalty = self._detect_trend(history, t)
        if trend_penalty > 0:
            factors.append(HealthFactor(
                name="temperature_trend",
                impact=-round(trend_penalty, 1),
                details=f"Temperature rising {trend_penalty:.1f}°C over window",
            ))
            score -= trend_penalty

        score = max(0, min(100, score))

        if score > 80:
            level = "healthy"
        elif score > 50:
            level = "warning"
        else:
            level = "critical"

        result.health_score = score
        result.health_level = level
        result.health_factors = factors
        return result

    def _detect_trend(self, history: list, current_temp: float) -> float:
        if len(history) < 2:
            return 0.0
        temps = [p.get("temperature", current_temp) for p in history if isinstance(p, dict)]
        temps = temps[-9:]
        temps.append(current_temp)
        if len(temps) < 2:
            return 0.0
        rate = (temps[-1] - temps[0]) / len(temps)
        if rate > self.trend_rate_threshold:
            return min(rate * 2, 15)
        return 0.0
