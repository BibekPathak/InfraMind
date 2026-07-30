"""Maintenance recommendation engine.

Rule-based: given health score, anomalies, and failure predictions,
suggests prioritized maintenance actions.
"""

from engines.base import BaseEngine, TelemetryInput, AnalysisResult, Recommendation


class MaintenanceRecommendationEngine(BaseEngine):
    name = "maintenance_recommendation"

    def analyze(self, telemetry: TelemetryInput) -> AnalysisResult:
        result = AnalysisResult()
        recommendations = []
        t = telemetry.temperature
        c = telemetry.current
        v = telemetry.voltage
        history = telemetry.history

        temp_trend = self._get_trend(history, "temperature")
        current_trend = self._get_trend(history, "current")

        if t > 90 and c > 150:
            recommendations.append(Recommendation(
                priority="critical",
                action="Schedule emergency inspection",
                reason="Both temperature and current critically high — possible internal fault",
                estimated_cost="high",
            ))
        elif t > 80 and temp_trend > 0.5 and c < 130:
            recommendations.append(Recommendation(
                priority="high",
                action="Inspect cooling system",
                reason="Temperature rising with normal load — cooling system may be failing",
                estimated_cost="medium",
            ))
        elif t > 80:
            recommendations.append(Recommendation(
                priority="high",
                action="Reduce load or increase cooling",
                reason="Temperature above threshold — check fans, radiators, and ambient temperature",
                estimated_cost="low",
            ))

        if c > 180:
            recommendations.append(Recommendation(
                priority="high",
                action="Check load balance and power quality",
                reason="Current draw significantly above normal — possible overload or imbalance",
                estimated_cost="low",
            ))
        elif c > 140 and current_trend > 1.0:
            recommendations.append(Recommendation(
                priority="medium",
                action="Monitor load trend",
                reason="Current steadily increasing — schedule winding resistance test",
                estimated_cost="medium",
            ))

        if v > 0 and v < 10000:
            recommendations.append(Recommendation(
                priority="high",
                action="Check voltage regulation equipment",
                reason="Voltage below minimum operating range — possible tap changer or regulator issue",
                estimated_cost="medium",
            ))

        if t < 30 and c == 0:
            recommendations.append(Recommendation(
                priority="critical",
                action="Verify sensor operation",
                reason="All sensor readings near zero — possible sensor failure or power loss",
                estimated_cost="low",
            ))

        if len(recommendations) == 0:
            if t > 70:
                recommendations.append(Recommendation(
                    priority="low",
                    action="Schedule routine inspection",
                    reason="Temperature elevated but within acceptable range",
                    estimated_cost="low",
                ))
            else:
                recommendations.append(Recommendation(
                    priority="low",
                    action="Continue normal monitoring",
                    reason="All parameters within normal range",
                    estimated_cost="none",
                ))

        result.recommendations = recommendations
        return result

    def _get_trend(self, history: list, metric: str) -> float:
        values = [
            p[metric] for p in history
            if isinstance(p, dict) and metric in p and isinstance(p[metric], (int, float))
        ]
        if len(values) < 3:
            return 0.0
        n = len(values)
        indices = list(range(n))
        x_mean = (n - 1) / 2
        y_mean = sum(values) / n
        num = sum((i - x_mean) * (v - y_mean) for i, v in zip(indices, values))
        den = sum((i - x_mean) ** 2 for i in indices)
        if den == 0:
            return 0.0
        return num / den
