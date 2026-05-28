package ingest

// CSVFileMapping maps a CSV filename to its target raw table.
type CSVFileMapping struct {
	CSVFile   string
	TableName string
	Required  bool
}

// AllTableMappings returns all CSV-to-raw-table mappings for the Olist dataset.
func AllTableMappings() []CSVFileMapping {
	return []CSVFileMapping{
		{CSVFile: "olist_customers_dataset.csv", TableName: "raw.olist_customers", Required: true},
		{CSVFile: "olist_orders_dataset.csv", TableName: "raw.olist_orders", Required: true},
		{CSVFile: "olist_order_items_dataset.csv", TableName: "raw.olist_order_items", Required: true},
		{CSVFile: "olist_order_payments_dataset.csv", TableName: "raw.olist_order_payments", Required: true},
		{CSVFile: "olist_order_reviews_dataset.csv", TableName: "raw.olist_order_reviews", Required: true},
		{CSVFile: "olist_products_dataset.csv", TableName: "raw.olist_products", Required: true},
		{CSVFile: "olist_sellers_dataset.csv", TableName: "raw.olist_sellers", Required: true},
		{CSVFile: "olist_geolocation_dataset.csv", TableName: "raw.olist_geolocation", Required: true},
		{CSVFile: "product_category_name_translation.csv", TableName: "raw.product_category_name_translation", Required: true},
		{CSVFile: "olist_marketing_qualified_leads_dataset.csv", TableName: "raw.marketing_qualified_leads", Required: false},
		{CSVFile: "olist_closed_deals_dataset.csv", TableName: "raw.closed_deals", Required: false},
	}
}
