import React from 'react';

interface FormattedTextProps {
  text: string;
  className?: string;
}

/**
 * Component to render formatted text with basic markdown-like syntax
 * Supports **bold** text formatting and emoji recognition for Claude log output
 */
export default function FormattedText({ text, className = '' }: FormattedTextProps) {
  // Simple function to render text with **bold** markdown and emoji styling
  const renderFormattedText = (text: string) => {
    const parts = text.split(/(\*\*.*?\*\*)/g);

    return parts.map((part, index) => {
      if (part.startsWith('**') && part.endsWith('**')) {
        // Remove the ** markers and render as bold
        const boldText = part.slice(2, -2);
        return (
          <strong key={index} className="font-semibold text-blue-300">
            {boldText}
          </strong>
        );
      } else {
        return <span key={index}>{part}</span>;
      }
    });
  };

  // Add specific styling based on message content/emojis
  const getMessageStyling = (text: string): string => {
    if (text.startsWith('🤖')) {
      return 'text-green-300'; // Assistant messages in green
    } else if (text.startsWith('👤')) {
      return 'text-blue-300'; // User messages in blue
    } else if (text.startsWith('🛠️')) {
      return 'text-purple-300'; // Tool use in purple
    } else if (text.startsWith('⚙️')) {
      return 'text-gray-300'; // System messages in gray
    } else if (text.startsWith('📊')) {
      return 'text-yellow-300'; // Results in yellow
    } else if (text.startsWith('🚀')) {
      return 'text-green-400'; // Started messages in bright green
    } else if (text.startsWith('🏁')) {
      return 'text-red-300'; // Finished messages in red
    } else if (text.startsWith('⏳')) {
      return 'text-orange-300'; // Thinking messages in orange
    } else if (text.startsWith('✅')) {
      return 'text-green-400'; // Success messages in bright green
    } else if (text.startsWith('❌')) {
      return 'text-red-400'; // Error messages in bright red
    } else if (text.startsWith('❓')) {
      return 'text-gray-400'; // Unknown messages in gray
    }
    return 'text-gray-100'; // Default text color
  };

  const messageStyling = getMessageStyling(text);

  return (
    <div className={`whitespace-pre-wrap break-all ${messageStyling} ${className}`}>
      {renderFormattedText(text)}
    </div>
  );
}