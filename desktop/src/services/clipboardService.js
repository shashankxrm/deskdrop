import { clipboard } from 'electron';

export const setupClipboardService = () => {
  return {
    writeText: (text) => {
      try {
        clipboard.writeText(text);
        console.log('Text written to clipboard:', text);
        return true;
      } catch (error) {
        console.error('Error writing to clipboard:', error);
        return false;
      }
    },
    
    readText: () => {
      try {
        return clipboard.readText();
      } catch (error) {
        console.error('Error reading clipboard:', error);
        return null;
      }
    }
  };
};

