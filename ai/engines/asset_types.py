"""Asset type registry for AI engines.

Defines per-asset-type thresholds and health scoring weights.
Matching the seeded asset_types in the backend (migrations 010).
"""

from dataclasses import dataclass, field
from typing import Dict


@dataclass
class AssetTypeConfig:
    type: str
    display_name: str
    temp_warning: float = 75
    temp_critical: float = 90
    current_warning: float = 120
    current_critical: float = 180
    voltage_min: float = 10000
    humidity_max: float = 80
    weights: Dict[str, float] = field(default_factory=lambda: {
        "temperature": 0.4,
        "current": 0.3,
        "voltage": 0.15,
        "humidity": 0.15,
    })


# Default (transformer-like) config used when type is unknown
DEFAULT_CONFIG = AssetTypeConfig(type="default", display_name="Default")

CONFIGS: Dict[str, AssetTypeConfig] = {
    "transformer": AssetTypeConfig(
        type="transformer",
        display_name="Transformer",
        temp_warning=75, temp_critical=90,
        current_warning=120, current_critical=180,
        voltage_min=10000, humidity_max=80,
        weights={"temperature": 0.4, "current": 0.3, "voltage": 0.15, "humidity": 0.15},
    ),
    "pump": AssetTypeConfig(
        type="pump",
        display_name="Pump",
        temp_warning=80, temp_critical=95,
        current_warning=200, current_critical=250,
        voltage_min=0, humidity_max=85,
        weights={"temperature": 0.3, "current": 0.3, "voltage": 0.2, "humidity": 0.2},
    ),
    "motor": AssetTypeConfig(
        type="motor",
        display_name="Motor",
        temp_warning=85, temp_critical=100,
        current_warning=110, current_critical=150,
        voltage_min=0, humidity_max=85,
        weights={"temperature": 0.3, "current": 0.3, "voltage": 0.2, "humidity": 0.2},
    ),
    "generator": AssetTypeConfig(
        type="generator",
        display_name="Generator",
        temp_warning=80, temp_critical=95,
        current_warning=200, current_critical=280,
        voltage_min=11000, humidity_max=80,
        weights={"temperature": 0.35, "current": 0.3, "voltage": 0.15, "humidity": 0.2},
    ),
    "hvac": AssetTypeConfig(
        type="hvac",
        display_name="HVAC System",
        temp_warning=35, temp_critical=45,
        current_warning=100, current_critical=150,
        voltage_min=0, humidity_max=70,
        weights={"temperature": 0.3, "current": 0.3, "voltage": 0.2, "humidity": 0.2},
    ),
}


def get_config(asset_type: str) -> AssetTypeConfig:
    if not asset_type:
        return DEFAULT_CONFIG
    return CONFIGS.get(asset_type, DEFAULT_CONFIG)
