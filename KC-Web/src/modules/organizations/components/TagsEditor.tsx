/**
 * Tags Editor Component
 *
 * Key-value tag editor for organizations.
 */

import React, { useState } from 'react';

import { Box, Chip, IconButton, Stack, TextField } from '@mui/material';

import { TAGS_EDITOR } from '@constants/messages';
import { AddIcon, DeleteIcon } from '@theme/icons';

interface TagsEditorProps {
  tags: Record<string, string>;
  onChange: (tags: Record<string, string>) => void;
}

const TagsEditor: React.FC<TagsEditorProps> = ({ tags, onChange }) => {
  const [newKey, setNewKey] = useState('');
  const [newValue, setNewValue] = useState('');

  const handleAdd = () => {
    if (newKey && !tags[newKey]) {
      onChange({ ...tags, [newKey]: newValue });
      setNewKey('');
      setNewValue('');
    }
  };

  const handleRemove = (keyToRemove: string) => {
    const rest = Object.fromEntries(Object.entries(tags).filter(([k]) => k !== keyToRemove));
    onChange(rest);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && newKey && !tags[newKey]) {
      e.preventDefault();
      handleAdd();
    }
  };

  return (
    <Box>
      <Stack direction="row" spacing={1} sx={{ mb: 1 }}>
        <TextField
          size="small"
          placeholder={TAGS_EDITOR.PLACEHOLDER_KEY}
          value={newKey}
          onChange={(e) => setNewKey(e.target.value)}
          onKeyDown={handleKeyDown}
          sx={{ width: 120 }}
        />
        <TextField
          size="small"
          placeholder={TAGS_EDITOR.PLACEHOLDER_VALUE}
          value={newValue}
          onChange={(e) => setNewValue(e.target.value)}
          onKeyDown={handleKeyDown}
          sx={{ flex: 1 }}
        />
        <IconButton onClick={handleAdd} disabled={!newKey || !!tags[newKey]} size="small">
          <AddIcon />
        </IconButton>
      </Stack>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
        {Object.entries(tags).map(([key, value]) => (
          <Chip
            key={key}
            label={`${key}: ${value}`}
            onDelete={() => handleRemove(key)}
            deleteIcon={<DeleteIcon />}
            size="small"
          />
        ))}
        {Object.keys(tags).length === 0 && (
          <Box sx={{ color: 'text.secondary', fontSize: '0.875rem' }}>
            {TAGS_EDITOR.LABEL_NO_TAGS}
          </Box>
        )}
      </Box>
    </Box>
  );
};

export default TagsEditor;
