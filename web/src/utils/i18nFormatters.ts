/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useTranslation } from 'react-i18next';

export const useLocalizedFormatters = () => {
  const { i18n } = useTranslation();
  
  const formatDate = (date: Date | string) => {
    const dateObj = typeof date === 'string' ? new Date(date) : date;
    return dateObj.toLocaleDateString(i18n.language);
  };
  
  const formatDateTime = (date: Date | string) => {
    const dateObj = typeof date === 'string' ? new Date(date) : date;
    return dateObj.toLocaleString(i18n.language);
  };
  
  const formatNumber = (num: number) => {
    return num.toLocaleString(i18n.language);
  };
  
  const formatDecimal = (num: number, decimals: number = 2) => {
    return num.toLocaleString(i18n.language, {
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
    });
  };
  
  return { formatDate, formatDateTime, formatNumber, formatDecimal };
};
