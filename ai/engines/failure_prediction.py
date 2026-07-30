"""Failure prediction engine using linear trend extrapolation.

Estimates time-to-warning and time-to-critical thresholds
based on the rate of change in temperature and current.
All math is pure Python — no external ML dependencies.
"""

import math
from engines.base import BaseEngine, TelemetryInput, AnalysisResult, FailurePrediction


class FailurePredictionEngine(BaseEngine):
    name = "failure_prediction"

    TEMP_WARNING = 80
    TEMP_CRITICAL = 90
    CURRENT_WARNING = 180
    CURRENT_CRITICAL = 220

    def analyze(self, telemetry: TelemetryInput) -> AnalysisResult:
        result = AnalysisResult()

        temp_slope, temp_conf = self._linear_trend(telemetry.history, "temperature")
        current_slope, current_conf = self._linear_trend(telemetry.history, "current")

        predictions = []

        if temp_slope > 0.1:
            tw = self._hours_to_threshold(telemetry.temperature, temp_slope, self.TEMP_WARNING)
            tc = self._hours_to_threshold(telemetry.temperature, temp_slope, self.TEMP_CRITICAL)
            predictions.append(("temperature", tw, tc, temp_conf, "rising"))

        if current_slope > 0.5:
            cw = self._hours_to_threshold(telemetry.current, current_slope, self.CURRENT_WARNING)
            cc = self._hours_to_threshold(telemetry.current, current_slope, self.CURRENT_CRITICAL)
            predictions.append(("current", cw, cc, current_conf, "rising"))

        if not predictions:
            result.failure_prediction = FailurePrediction(
                confidence=0.95, trend_direction="stable"
            )
            return result

        pred = predictions[0]
        result.failure_prediction = FailurePrediction(
            time_to_warning_hours=pred[1] if pred[1] < 9999 else None,
            time_to_critical_hours=pred[2] if pred[2] < 9999 else None,
            confidence=round(pred[3], 2),
            trend_direction=pred[4],
        )

        return result

    def _linear_trend(self, history: list, metric: str):
        values = [
            p[metric] for p in history
            if isinstance(p, dict) and metric in p and isinstance(p[metric], (int, float))
        ]
        if len(values) < 3:
            return 0, 0

        n = len(values)
        indices = list(range(n))
        x_mean = (n - 1) / 2
        y_mean = sum(values) / n

        num = sum((i - x_mean) * (v - y_mean) for i, v in zip(indices, values))
        den = sum((i - x_mean) ** 2 for i in indices)

        if den == 0:
            return 0, 0

        slope = num / den
        intercept = y_mean - slope * x_mean

        ss_res = sum((v - (slope * i + intercept)) ** 2 for i, v in zip(indices, values))
        ss_tot = sum((v - y_mean) ** 2 for v in values)

        r_squared = 1 - (ss_res / ss_tot) if ss_tot > 0 else 0

        confidence = max(0, min(1, r_squared * 0.8 + 0.2))
        return slope, confidence

    def _hours_to_threshold(self, current_value: float, slope: float, threshold: float) -> float:
        if slope <= 0:
            return 9999
        steps = (threshold - current_value) / slope
        if steps <= 0:
            return 0
        return steps * 2 / 60
