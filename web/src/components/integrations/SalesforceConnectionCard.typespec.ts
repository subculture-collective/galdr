import type { ComponentType } from "react";
import SalesforceConnectionCard from "./SalesforceConnectionCard";
import type { SalesforceStatus } from "@/lib/salesforce";

const card: ComponentType = SalesforceConnectionCard;

const connectedStatus: SalesforceStatus = {
  status: "active",
  external_account_id: "00Dxx0000001gPFEAY",
  instance_url: "https://acme.my.salesforce.com",
  account_count: 12,
  contact_count: 340,
  opportunity_count: 28,
  last_sync_at: "2026-05-05T12:00:00Z",
};

void card;
void connectedStatus;
