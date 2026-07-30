"""Anomaly detection engine using statistical methods.

Detects anomalies via:
- Moving average deviation
- Z-score outlier detection
- Rate-of-change (sudden spike) detection
"""

import math
from engines.base import BaseEngine, TelemetryInput, AnalysisResult, Anomaly


class AnomalyEngine(BaseEngine):
    name = "anomaly"

    def analyze(self, telemetry: TelemetryInput) -> AnalysisResult:
        result = AnalysisResult()
        anomalies = []

        anomalies.extend(self._check_zscore(telemetry, "temperature", telemetry.temperature))
        anomalies.extend(self._check_zscore(telemetry, "current", telemetry.current))
        anomalies.extend(self._check_zscore(telemetry, "voltage", telemetry.voltage))
        anomalies.extend(self._check_zscore(telemetry, "humidity", telemetry.humidity))

        anomalies.extend(self._check_rate_of_change(telemetry, "temperature", telemetry.temperature, 5.0))
        anomalies.extend(self._check_rate_of_change(telemetry, "current", telemetry.current, 30.0))

        if telemetry.current == 0 and telemetry.temperature < 30:
            anomalies.append(Anomaly(
                metric="current",
                severity="critical",
                description="Current reading is zero — possible sensor failure",
                value=0,
                expected=100,
            ))

        result.anomalies = anomalies
        return result

    def _get_history_values(self, telemetry: TelemetryInput, metric: str) -> list:
        values = []
        for p in telemetry.history:
            if isinstance(p, dict) and metric in p:
                v = p[metric]
                if isinstance(v, (int, float)):
                    values.append(v)
        return values

    def _mean(self, values: list) -> float:
        if not values:
            return 0.0
        return sum(values) / len(values)

    def _stdev(self, values: list, mean: float) -> float:
        if len(values) < 2:
            return 0.0
        variance = sum((v - mean) ** 2 for v in values) / len(values)
        return math.sqrt(variance)

    def _check_zscore(self, telemetry: TelemetryInput, metric: str, current: float) -> list:
        anomalies = []
        history = self._get_history_values(telemetry, metric)
        if len(history) < 3:
            return anomalies

        mean = self._mean(history)
        std = self._stdev(history, mean)
        if std == 0:
            return anomalies

        z = abs(current - mean) / std
        if z > 4:
            anomalies.append(Anomaly(
                metric=metric,
                severity="critical",
                description=f"{metric.capitalize()} z-score {z:.1f} — extreme outlier",
                value=current,
                expected=round(mean, 1),
            ))
        elif z > 3:
            anomalies.append(Anomaly(
                metric=metric,
                severity="warning",
                description=f"{metric.capitalize()} z-score {z:.1f} — outlier",
                value=current,
                expected=round(mean, 1),
            ))
        elif z > 2:
            anomalies.append(Anomaly(
                metric=metric,
                severity="info",
                description=f"{metric.capitalize()} z-score {z:.1f} — unusual",
                value=current,
                expected=round(mean, 1),
            ))

        return anomalies

    def _check_rate_of_change(self, telemetry: TelemetryInput, metric: str, current: float, threshold: float) -> list:
        anomalies = []
        history = self._get_history_values(telemetry, metric)
        if len(history) < 2:
            return anomalies

        prev = history[-2] if len(history) > 1 else history[-1]
        delta = abs(current - prev)

        if delta > threshold * 2:
            anomalies.append(Anomaly(
                metric=metric,
                severity="critical",
                description=f"Sudden {metric} spike of {delta:.1f} — possible event",
                value=current,
                expected=prev,
            ))
        elif delta > threshold:
            anomalies.append(Anomaly(
                metric=metric,
                severity="warning",
                description=f"Significant {metric} change of {delta:.1f}",
                value=current,
                expected=prev,
            ))

        return anomalies
