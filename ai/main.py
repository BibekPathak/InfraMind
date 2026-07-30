"""InfraMind AI Service — Multi-engine analytics platform.

Runs a pipeline of deterministic engines to analyze telemetry data.
No ML dependencies — all reasoning is transparent and explainable.
"""

from typing import List
from pydantic import BaseModel

from fastapi import FastAPI

from engines.base import (
    TelemetryInput,
    HealthFactor,
    Anomaly,
    FailurePrediction,
    Recommendation,
    AnalysisResult,
)
from engines.health_score import HealthScoreEngine


class HealthRequest(BaseModel):
    temperature: float = 0
    current: float = 0
    voltage: float = 0
    humidity: float = 0
    history: List[dict] = []


class FactorModel(BaseModel):
    name: str
    impact: float
    details: str = ""


class AnomalyModel(BaseModel):
    metric: str
    severity: str
    description: str
    value: float
    expected: float


class FailurePredictionModel(BaseModel):
    time_to_warning_hours: float | None = None
    time_to_critical_hours: float | None = None
    confidence: float = 0
    trend_direction: str = "stable"


class RecommendationModel(BaseModel):
    priority: str
    action: str
    reason: str
    estimated_cost: str = "unknown"


class HealthResponse(BaseModel):
    score: float
    level: str
    factors: List[FactorModel]


class AnalysisResponse(BaseModel):
    health_score: float
    health_level: str
    health_factors: List[FactorModel]
    anomalies: List[AnomalyModel]
    failure_prediction: FailurePredictionModel | None = None
    recommendations: List[RecommendationModel]


app = FastAPI(title="InfraMind AI", version="0.2.0")


def _to_telemetry(req: HealthRequest) -> TelemetryInput:
    return TelemetryInput(
        temperature=req.temperature,
        current=req.current,
        voltage=req.voltage,
        humidity=req.humidity,
        history=req.history,
    )


def _to_health_response(result: AnalysisResult) -> HealthResponse:
    return HealthResponse(
        score=result.health_score,
        level=result.health_level,
        factors=[FactorModel(name=f.name, impact=f.impact, details=f.details)
                 for f in result.health_factors],
    )


def _to_analysis_response(result: AnalysisResult) -> AnalysisResponse:
    fp = result.failure_prediction
    pred_model = None
    if fp:
        pred_model = FailurePredictionModel(
            time_to_warning_hours=fp.time_to_warning_hours,
            time_to_critical_hours=fp.time_to_critical_hours,
            confidence=fp.confidence,
            trend_direction=fp.trend_direction,
        )

    return AnalysisResponse(
        health_score=result.health_score,
        health_level=result.health_level,
        health_factors=[FactorModel(name=f.name, impact=f.impact, details=f.details)
                        for f in result.health_factors],
        anomalies=[AnomalyModel(
            metric=a.metric, severity=a.severity,
            description=a.description, value=a.value, expected=a.expected,
        ) for a in result.anomalies],
        failure_prediction=pred_model,
        recommendations=[RecommendationModel(
            priority=r.priority, action=r.action,
            reason=r.reason, estimated_cost=r.estimated_cost,
        ) for r in result.recommendations],
    )


@app.get("/health")
async def health():
    return {"status": "ok", "service": "infra-ai", "version": "0.2.0"}


@app.post("/health-score", response_model=HealthResponse)
async def health_score(req: HealthRequest):
    engine = HealthScoreEngine()
    result = engine.analyze(_to_telemetry(req))
    return _to_health_response(result)


@app.post("/analyze", response_model=AnalysisResponse)
async def analyze(req: HealthRequest):
    telemetry = _to_telemetry(req)
    result = AnalysisResult()

    engine = HealthScoreEngine()
    r = engine.analyze(telemetry)
    result.health_score = r.health_score
    result.health_level = r.health_level
    result.health_factors = r.health_factors

    return _to_analysis_response(result)
