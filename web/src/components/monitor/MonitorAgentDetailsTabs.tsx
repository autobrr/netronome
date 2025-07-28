/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import { MonitorAgent } from "@/api/monitor";
import { MonitorOverviewTab } from "./tabs/MonitorOverviewTab";
import { MonitorBandwidthTab } from "./tabs/MonitorBandwidthTab";
import { MonitorSystemTab } from "./tabs/MonitorSystemTab";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

interface MonitorAgentDetailsTabsProps {
  agent: MonitorAgent;
}

export const MonitorAgentDetailsTabs: React.FC<
  MonitorAgentDetailsTabsProps
> = ({ agent }) => {
  return (
    <Tabs defaultValue="overview" className="space-y-2">
      <TabsList>
        <TabsTrigger value="overview">Overview</TabsTrigger>
        <TabsTrigger value="bandwidth">Bandwidth</TabsTrigger>
        <TabsTrigger value="system">System & Hardware</TabsTrigger>
      </TabsList>

      <TabsContent value="overview">
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.15 }}
        >
          <MonitorOverviewTab agent={agent} />
        </motion.div>
      </TabsContent>

      <TabsContent value="bandwidth">
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.15 }}
        >
          <MonitorBandwidthTab agent={agent} />
        </motion.div>
      </TabsContent>

      <TabsContent value="system">
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.15 }}
        >
          <MonitorSystemTab agent={agent} />
        </motion.div>
      </TabsContent>
    </Tabs>
  );
};
