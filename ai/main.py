from typing import List
from pydantic import BaseModel
from engines.health_score import calculate_health_score


class HealthRequest(BaseModel):
    temperature: float = 0
    current: float = 0
    voltage: float = 0
    humidity: float = 0


class HealthFactor(BaseModel):
    name: str
    impact: float


class HealthResponse(BaseModel):
    score: float
    level: str
    factors: List[HealthFactor]


from fastapi import FastAPI

app = FastAPI(title="InfraMind AI", version="0.1.0")


@app.get("/health")
async def health():
    return {"status": "ok", "service": "infra-ai"}


@app.post("/health-score", response_model=HealthResponse)
async def health_score(req: HealthRequest):
    score, level, factors = calculate_health_score(
        temperature=req.temperature,
        current=req.current,
        voltage=req.voltage,
        humidity=req.humidity,
    )
    return HealthResponse(
        score=score,
        level=level,
        factors=[HealthFactor(name=f.name, impact=f.impact) for f in factors],
    )
