import uuid
from typing import Optional
from pydantic import BaseModel, Field
from datetime import datetime


def gen_id(prefix: str = '') -> str:
    raw = uuid.uuid4().hex[:12]
    return f"{prefix}_{raw}" if prefix else raw


class StrategyRecommendation(BaseModel):
    recommendation_id: str = Field(default_factory=lambda: gen_id('rec'))
    event_id: str = Field(default='', description='Associated alert event ID')
    title: str = Field(min_length=1, description='Strategy title')
    detail: str = Field(min_length=1, description='Structured text with [问题][证据][判断][建议动作][预期收益][风险][验收指标]')
    target_object: str = Field(default='', description='Target object (seller, category, region, etc.)')
    expected_impact: str = Field(default='', description='Expected impact description')
    risk_level: str = Field(default='medium', description='high, medium, or low')
    requires_approval: bool = Field(default=False)
    owner_role: str = Field(default='business_ops')
    decision_type: str = Field(default='monitor_only', description='monitor_only, investigate, optimize, intervention, experiment')
    confidence: str = Field(default='medium', description='high, medium, or low')
    success_metric: str = Field(default='', description='Verification metric for retro')
    impact_score: int = Field(default=0, ge=0)
    created_at: str = Field(default_factory=lambda: datetime.now().isoformat())
    decision_source: str = Field(default="heuristic", description="Decision origin: 'heuristic' or 'llm'")
    model_name: Optional[str] = Field(default=None, description="LLM model name if decision_source='llm'")
    is_simulated: bool = Field(default=True, description="Whether this is a simulated/synthetic scenario")


class ActionTask(BaseModel):
    task_id: str = Field(default_factory=lambda: gen_id('task'))
    title: str = Field(min_length=1)
    description: str = Field(default='')
    owner_role: str = Field(default='business_ops')
    source_event: str = Field(default='')
    source_strategy: str = Field(default='')
    priority: str = Field(default='medium', description='high, medium, or low')
    deadline: str = Field(default='')
    status: str = Field(default='todo', description='todo, in_progress, blocked, done, cancelled')
    task_source: str = Field(default="heuristic", description="Task origin: 'heuristic' or 'llm'")
    source_rule: Optional[str] = Field(default=None, description="Rule ID that triggered this task")
    requires_human_confirmation: bool = Field(default=True, description="Whether human approval is needed")


class ReviewRetro(BaseModel):
    review_id: str = Field(default_factory=lambda: gen_id('rev'))
    strategy_id: str = Field(default='')
    outcome: str = Field(default='')
    actual_impact: str = Field(default='')
    is_effective: bool = Field(default=False)
    lessons_learned: str = Field(default='')
    promote_to_rule: bool = Field(default=False)
    reviewed_at: str = Field(default='')
    review_type: str = Field(default="simulated", description="Review type: 'simulated' or 'human'")
    review_source: str = Field(default="hindsight_rule", description="Review data source: 'hindsight_rule' or 'manual_feedback'")


class DecisionReport(BaseModel):
    generated_at: str = Field(default_factory=lambda: datetime.now().isoformat())
    mode: str = Field(default='full')
    total_alerts: int = Field(default=0)
    strategies_count: int = Field(default=0)
    tasks_count: int = Field(default=0)
    top_findings: list[str] = Field(default_factory=list)
    recommendations_summary: str = Field(default='')


STRATEGY_SECTIONS = ['【问题】', '【证据】', '【判断】', '【建议动作】', '【预期收益】', '【风险】', '【验收指标】']


def validate_strategy_detail(detail: str) -> tuple[bool, list[str]]:
    missing = [s for s in STRATEGY_SECTIONS if s not in detail]
    return len(missing) == 0, missing
