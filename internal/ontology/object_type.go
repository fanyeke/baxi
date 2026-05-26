package ontology

// All AIP object type identifiers.
const (
	TypeCustomer      = "customer"
	TypeOrder         = "order"
	TypeSeller        = "seller"
	TypeProduct       = "product"
	TypeCategory      = "category"
	TypeRegion        = "region"
	TypeMarketingLead = "marketing_lead"
	TypeMetricAlert   = "metric_alert"
)

// AllObjectTypes returns the complete list of known AIP object type names.
// Order is stable: customer → order → seller → product → category → region →
// marketing_lead → metric_alert.
func AllObjectTypes() []string {
	return []string{
		TypeCustomer,
		TypeOrder,
		TypeSeller,
		TypeProduct,
		TypeCategory,
		TypeRegion,
		TypeMarketingLead,
		TypeMetricAlert,
	}
}

// ObjectTypeDisplayNames returns a human-readable (Chinese) display name for
// each object type, as defined in aip_object_schema.yml.
func ObjectTypeDisplayName(objectType string) string {
	switch objectType {
	case TypeCustomer:
		return "客户"
	case TypeOrder:
		return "订单"
	case TypeSeller:
		return "卖家"
	case TypeProduct:
		return "产品"
	case TypeCategory:
		return "品类"
	case TypeRegion:
		return "区域"
	case TypeMarketingLead:
		return "营销线索"
	case TypeMetricAlert:
		return "异常事件"
	default:
		return objectType
	}
}

// KnownObjectType checks whether the given name is a known AIP object type.
func KnownObjectType(name string) bool {
	switch name {
	case TypeCustomer, TypeOrder, TypeSeller, TypeProduct,
		TypeCategory, TypeRegion, TypeMarketingLead, TypeMetricAlert:
		return true
	default:
		return false
	}
}
