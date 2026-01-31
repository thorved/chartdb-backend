/**
 * Diagram Export/Import Utilities
 * Duplicates ChartDB's export-import-utils.ts logic
 * Maintains independence from ChartDB internal structure
 */

/**
 * Generate a new diagram ID
 */
export function generateDiagramId() {
    return 'diagram_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9)
}

/**
 * Generate a new ID for any entity
 */
export function generateId() {
    return Date.now().toString(36) + Math.random().toString(36).substr(2, 5)
}

/**
 * Running ID generator for cloning
 */
export function createRunningIdGenerator() {
    let id = 0
    return () => (id++).toString()
}

/**
 * Clone a diagram with new IDs
 * This mirrors ChartDB's cloneDiagram function
 * @param {Object} diagram - The diagram to clone
 * @param {Object} options - Options for cloning
 * @param {Function} options.generateId - Custom ID generator function
 * @param {boolean} options.preserveDiagramId - If true, keeps the original diagram ID (useful for sync)
 */
export function cloneDiagram(diagram, options = {}) {
    const idGenerator = options.generateId || generateId
    const preserveDiagramId = options.preserveDiagramId || false
    const idMap = new Map()

    // Clone the diagram (preserve ID if syncing, generate new if importing)
    const clonedDiagram = {
        ...diagram,
        id: preserveDiagramId ? diagram.id : generateDiagramId(),
        createdAt: Date.now(),
        updatedAt: Date.now(),
    }

    // Map old diagram ID to new
    idMap.set(diagram.id, clonedDiagram.id)

    // Clone tables with new IDs
    if (diagram.tables) {
        clonedDiagram.tables = diagram.tables.map(table => {
            const newTableId = idGenerator()
            idMap.set(table.id, newTableId)

            const clonedTable = {
                ...table,
                id: newTableId,
                createdAt: table.createdAt || Date.now(),
            }

            // Clone fields with new IDs
            if (table.fields) {
                clonedTable.fields = table.fields.map(field => {
                    const newFieldId = idGenerator()
                    idMap.set(field.id, newFieldId)
                    return {
                        ...field,
                        id: newFieldId,
                        createdAt: field.createdAt || Date.now(),
                    }
                })
            }

            // Clone indexes with updated field references
            if (table.indexes) {
                clonedTable.indexes = table.indexes.map(index => ({
                    ...index,
                    id: idGenerator(),
                    fieldIds: index.fieldIds?.map(oldId => idMap.get(oldId) || oldId) || [],
                    createdAt: index.createdAt || Date.now(),
                }))
            }

            return clonedTable
        })
    }

    // Clone relationships with updated references
    if (diagram.relationships) {
        clonedDiagram.relationships = diagram.relationships.map(rel => ({
            ...rel,
            id: idGenerator(),
            sourceTableId: idMap.get(rel.sourceTableId) || rel.sourceTableId,
            targetTableId: idMap.get(rel.targetTableId) || rel.targetTableId,
            sourceFieldId: idMap.get(rel.sourceFieldId) || rel.sourceFieldId,
            targetFieldId: idMap.get(rel.targetFieldId) || rel.targetFieldId,
            createdAt: rel.createdAt || Date.now(),
        }))
    }

    // Clone dependencies with updated references
    if (diagram.dependencies) {
        clonedDiagram.dependencies = diagram.dependencies.map(dep => ({
            ...dep,
            id: idGenerator(),
            tableId: idMap.get(dep.tableId) || dep.tableId,
            dependentTableId: idMap.get(dep.dependentTableId) || dep.dependentTableId,
            createdAt: dep.createdAt || Date.now(),
        }))
    }

    // Clone areas
    if (diagram.areas) {
        clonedDiagram.areas = diagram.areas.map(area => ({
            ...area,
            id: idGenerator(),
        }))
    }

    // Clone notes
    if (diagram.notes) {
        clonedDiagram.notes = diagram.notes.map(note => ({
            ...note,
            id: idGenerator(),
        }))
    }

    // Clone custom types
    if (diagram.customTypes) {
        clonedDiagram.customTypes = diagram.customTypes.map(ct => ({
            ...ct,
            id: idGenerator(),
        }))
    }

    return { diagram: clonedDiagram, idMap }
}

/**
 * Convert diagram to JSON output (for export/server storage)
 * Mirrors ChartDB's diagramToJSONOutput
 */
export function diagramToJSON(diagram) {
    const { diagram: clonedDiagram } = cloneDiagram(diagram, {
        generateId: createRunningIdGenerator()
    })
    return JSON.stringify(clonedDiagram, null, 2)
}

/**
 * Parse diagram from JSON input (for import/server retrieval)
 * Mirrors ChartDB's diagramFromJSONInput
 */
export function diagramFromJSON(jsonString) {
    let loadedDiagram

    try {
        loadedDiagram = JSON.parse(jsonString)
    } catch (e) {
        throw new Error('Invalid JSON format')
    }

    // Validate required fields
    if (!loadedDiagram.id || !loadedDiagram.name) {
        throw new Error('Invalid diagram format: missing required fields')
    }

    // Clone with new IDs (ChartDB always assigns new IDs on import)
    const { diagram } = cloneDiagram(loadedDiagram)

    return diagram
}

/**
 * Prepare diagram data from IndexedDB for server push
 * Takes raw IndexedDB data and formats it for API
 */
export function prepareDiagramForServer(diagramData) {
    // Ensure all arrays exist
    return {
        id: diagramData.id,
        name: diagramData.name || 'Untitled',
        databaseType: diagramData.databaseType || 'generic',
        databaseEdition: diagramData.databaseEdition,
        tables: diagramData.tables || [],
        relationships: diagramData.relationships || [],
        dependencies: diagramData.dependencies || [],
        areas: diagramData.areas || [],
        notes: diagramData.notes || [],
        customTypes: diagramData.customTypes || [],
        createdAt: diagramData.createdAt,
        updatedAt: diagramData.updatedAt || Date.now(),
    }
}

/**
 * Prepare diagram data from server for IndexedDB storage
 */
export function prepareDiagramForIndexedDB(diagramData) {
    return {
        id: diagramData.id,
        name: diagramData.name || 'Untitled',
        databaseType: diagramData.databaseType || 'generic',
        databaseEdition: diagramData.databaseEdition,
        tables: diagramData.tables || [],
        relationships: diagramData.relationships || [],
        dependencies: diagramData.dependencies || [],
        areas: diagramData.areas || [],
        notes: diagramData.notes || [],
        customTypes: diagramData.customTypes || [],
        createdAt: diagramData.createdAt || Date.now(),
        updatedAt: diagramData.updatedAt || Date.now(),
    }
}
