  provider "azurerm" {
  subscription_id = var.subscription_id 
  client_id       = var.client_id
  client_secret   = var.client_secret
  tenant_id       = var.tenant_id
  features {}
}

resource "azurerm_resource_group" "joerod-dev" {
    name     = "joerod-dev"
    location = "East US"
}

resource "azurerm_monitor_action_group" "joerod" {
  name                = "joerod"
  resource_group_name = azurerm_resource_group.joerod-dev.name
  short_name          = "joerod"

  email_receiver {
    name                    = "JoeRod"
    email_address           = "joerod@gmail.com"
    use_common_alert_schema = true
  }
}

resource "azurerm_log_analytics_workspace" "joerod" {
  name                = "joerod"
  location            = azurerm_resource_group.joerod-dev.location
  resource_group_name = azurerm_resource_group.joerod-dev.name
}

resource "azurerm_monitor_scheduled_query_rules_alert" "joerod" {
  name                = "Heartbeat"
  location            = azurerm_resource_group.joerod-dev.location
  resource_group_name = azurerm_resource_group.joerod-dev.name
  action {
    action_group           = [azurerm_monitor_action_group.joerod.id]
  }
  data_source_id = azurerm_log_analytics_workspace.joerod.id
  description    = "Heartbeat"
  enabled        = true
  query       = <<-QUERY
    Heartbeat | summarize LastCall = max(TimeGenerated) by Computer | where LastCall < ago(5m)
  QUERY
  severity    = 2
  frequency   = 5
  time_window = 1440
  trigger {
    operator  = "GreaterThan"
    threshold = 0
  }
}