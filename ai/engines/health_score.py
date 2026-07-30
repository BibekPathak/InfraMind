from engines.base import BaseEngine, TelemetryInput, AnalysisResult, HealthFactor


class HealthScoreEngine(BaseEngine):
    name = "health_score"

    def analyze(self, telemetry: TelemetryInput) -> AnalysisResult:
        t = telemetry.temperature
        c = telemetry.current
        v = telemetry.voltage
        h = telemetry.humidity

        result = AnalysisResult()
        score = 100.0
        factors = []

        temp_penalty = max(0, (t - 75) * 1.5)
        temp_penalty = min(temp_penalty, 40)
        factors.append(HealthFactor(name="temperature", impact=-round(temp_penalty, 1),
                                     details=f"{t}°C exceeds 75°C threshold"))
        score -= temp_penalty

        current_penalty = max(0, (c - 120) * 0.3)
        current_penalty = min(current_penalty, 30)
        factors.append(HealthFactor(name="current", impact=-round(current_penalty, 1),
                                     details=f"{c}A exceeds 120A threshold"))
        score -= current_penalty

        humidity_penalty = min(h * 0.2, 10)
        factors.append(HealthFactor(name="humidity", impact=-round(humidity_penalty, 1),
                                     details=f"{h}% humidity impact"))
        score -= humidity_penalty

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
