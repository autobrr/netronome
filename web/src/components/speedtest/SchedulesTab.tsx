/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import ScheduleManager from "./ScheduleManager";
import { Server } from "@/types/types";

interface SchedulesTabProps {
  servers: Server[];
  selectedServers: Server[];
  onServerSelect: (server: Server) => void;
}

export const SchedulesTab: React.FC<SchedulesTabProps> = ({
  servers,
  selectedServers,
  onServerSelect,
}) => {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
      className="w-full"
    >
      <ScheduleManager
        servers={servers}
        selectedServers={selectedServers}
        onServerSelect={onServerSelect}
      />
    </motion.div>
  );
};